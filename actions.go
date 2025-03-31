package main

import (
	"encoding/json"
	"fmt"
	"sync"
)

func (e *Episode) AddToQueue(action string) ([]*Episode, error) {
	// action: "play_next", "play_last", "play_now"
	if e.UUID == "" || e.PodcastUUID == "" || e.Title == "" || e.URL == "" {
		return nil, fmt.Errorf("episode info missing")
	}
	body := map[string]any{
		"version": 2,
		"episode": map[string]any{
			"uuid":    e.UUID,
			"podcast": e.PodcastUUID,
			"title":   e.Title,
			"url":     e.URL,
		},
	}
	if action == "play_last" {
		// if the episode is already in the queue, do nothing
		if upNext, err := GetUpNext(true); err == nil {
			for _, episode := range upNext {
				if episode.UUID == e.UUID {
					return upNext, nil
				}
			}
		}
	}
	var response PocketCastsUpNextResponse
	if err := PocketCastsRequest("/up_next/"+action, &body, &response); err != nil {
		return nil, err
	}
	return processUpNextResponse(&response)
}

func RemoveEpisodesFromQueue(episodes []*Episode) ([]*Episode, error) {
	if len(episodes) == 0 {
		return nil, fmt.Errorf("no episodes to remove")
	}
	uuidList := make([]string, len(episodes))
	for i, e := range episodes {
		if e.UUID == "" {
			return nil, fmt.Errorf("episode UUID not set")
		}
		uuidList[i] = e.UUID
	}
	body := map[string]any{
		"version": 2,
		"uuids":   uuidList,
	}
	var response PocketCastsUpNextResponse
	if err := PocketCastsRequest("/up_next/remove", &body, &response); err != nil {
		return nil, err
	}
	return processUpNextResponse(&response)
}

func ArchiveEpisodes(episodes []*Episode, markAsPlayed bool) error {
	if len(episodes) == 0 {
		return fmt.Errorf("no episodes to archive")
	}
	episodeList := make([]map[string]string, len(episodes))
	for i, e := range episodes {
		if e.UUID == "" || e.PodcastUUID == "" {
			return fmt.Errorf("episode info missing")
		}
		if markAsPlayed {
			e.Update(map[string]any{
				"status": 3,
			})
		}
		episodeList[i] = map[string]string{
			"uuid":    e.UUID,
			"podcast": e.PodcastUUID,
		}
	}
	body := map[string]any{
		"episodes": episodeList,
		"archive":  true,
	}

	var wg sync.WaitGroup
	var archiveErr, queueErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		archiveErr = PocketCastsRequest("/sync/update_episodes_archive", &body, nil)
	}()

	go func() {
		defer wg.Done()
		if _, err := RemoveEpisodesFromQueue(episodes); err != nil {
			queueErr = err
		}
	}()

	wg.Wait()

	if archiveErr != nil {
		return archiveErr
	}
	if queueErr != nil {
		return queueErr
	}
	return nil
}

func (e *Episode) Archive(markAsPlayed bool) error {
	return ArchiveEpisodes([]*Episode{e}, markAsPlayed)
}

func (e *Episode) Update(body map[string]any) error {
	// update position: {"position": "1234", "status": 2}
	// mark as played: {"status": 3}
	if e.UUID == "" || e.PodcastUUID == "" {
		return fmt.Errorf("episode info missing")
	}
	body["uuid"] = e.UUID
	body["podcast"] = e.PodcastUUID
	return PocketCastsRequest("/sync/update_episode", &body, nil)
}

func AddFeed(url string, pollUUID *string) (*Podcast, error) {
	body := map[string]any{
		"url":           url,
		"poll_uuid":     pollUUID,
		"public_option": "no",
	}
	var response struct {
		Status   string `json:"status"`
		PollUUID string `json:"poll_uuid"`
		Result   struct {
			Podcast struct {
				Name   string `json:"title"`
				Author string `json:"author"`
				Desc   string `json:"description"`
				Image  string `json:"thumbnail_url"`
				Link   string `json:"url"`
				UUID   string `json:"uuid"`
			} `json:"podcast"`
		} `json:"result"`
	}
	if err := PocketCastsRequest("refresh.pocketcasts.com/author/add_feed_url", &body, &response); err != nil {
		return nil, err
	}
	switch response.Status {
	case "poll":
		return AddFeed(url, &response.PollUUID)
	case "ok":
		return &Podcast{
			Name:   response.Result.Podcast.Name,
			Author: response.Result.Podcast.Author,
			Desc:   response.Result.Podcast.Desc,
			Image:  response.Result.Podcast.Image,
			Link:   response.Result.Podcast.Link,
			UUID:   response.Result.Podcast.UUID,
		}, nil
	default:
		return nil, fmt.Errorf("invalid feed URL")
	}
}

func (p *Podcast) Subscribe() error {
	if p.UUID == "" && p.URL != "" {
		if podcast, err := AddFeed(p.URL, nil); err != nil {
			return err
		} else {
			p.Name = podcast.Name
			p.Author = podcast.Author
			p.Desc = podcast.Desc
			p.Image = podcast.Image
			p.Link = podcast.Link
			p.UUID = podcast.UUID
		}
	}
	if p.UUID == "" {
		return fmt.Errorf("podcast UUID not set")
	}
	body := map[string]any{
		"uuid": p.UUID,
	}
	return PocketCastsRequest("/user/podcast/subscribe", &body, nil)
}

func (p *Podcast) Unsubscribe() error {
	if p.UUID == "" {
		return fmt.Errorf("podcast UUID not set")
	}
	body := map[string]any{
		"uuid": p.UUID,
	}
	return PocketCastsRequest("/user/podcast/unsubscribe", &body, nil)
}

func SearchPodcasts(term string) ([]*Podcast, error) {
	body := map[string]any{
		"term": term,
	}
	var response struct {
		Podcasts []struct {
			Name   string `json:"title"`
			Author string `json:"author"`
			Desc   string `json:"description"`
			Link   string `json:"url"`
			UUID   string `json:"uuid"`
		} `json:"podcasts"`
	}
	if err := PocketCastsRequest("/discover/search", &body, &response); err != nil {
		return nil, err
	}
	podcasts := make([]*Podcast, len(response.Podcasts))
	for i, podcast := range response.Podcasts {
		podcasts[i] = &Podcast{
			Name:   podcast.Name,
			Author: podcast.Author,
			Desc:   podcast.Desc,
			Link:   podcast.Link,
			UUID:   podcast.UUID,
			Image:  fmt.Sprintf("https://static.pocketcasts.com/discover/images/webp/200/%s.webp", podcast.UUID),
		}
	}
	file := getCachePath("search_results")
	data, _ := json.Marshal(podcasts)
	writeCache(file, data)
	return podcasts, nil
}
