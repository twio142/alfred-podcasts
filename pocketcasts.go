package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	tokenMutex       sync.Mutex
	pocketCastsToken string
)

func getToken() error {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	if pocketCastsToken != "" {
		return nil
	}
	if token, err := os.ReadFile(".token"); err != nil {
		return PocketCastsLogin(os.Getenv("email"), os.Getenv("password"))
	} else {
		pocketCastsToken = string(token)
		return nil
	}
}

type PocketCastsUpNextResponse struct {
	Episodes []struct {
		UUID        string    `json:"uuid"`
		Title       string    `json:"title"`
		URL         string    `json:"url"`
		PodcastUUID string    `json:"podcast"`
		Date        time.Time `json:"published"`
	} `json:"episodes"`
	EpisodeSync []struct {
		UUID       string `json:"uuid"`
		PlayedUpTo int    `json:"playedUpTo"`
		Duration   int    `json:"duration"`
	}
}

type PocketCastsPodcastsResponse struct {
	Podcasts []struct {
		UUID        string    `json:"uuid"`
		Name        string    `json:"title"`
		Author      string    `json:"author"`
		Link        string    `json:"url"`
		Desc        string    `json:"description"`
		LastUpdated time.Time `json:"lastEpisodePublished"`
	} `json:"podcasts"`
}

type PocketCastsEpisodesResponse struct {
	Podcast struct {
		UUID     string `json:"uuid"`
		Name     string `json:"title"`
		Author   string `json:"author"`
		Link     string `json:"url"`
		Desc     string `json:"description"`
		Episodes []struct {
			UUID      string    `json:"uuid"`
			Title     string    `json:"title"`
			URL       string    `json:"url"`
			ShowNotes string    `json:"show_notes"`
			Image     string    `json:"image"`
			Date      time.Time `json:"published"`
			Duration  int       `json:"duration"`
		}
	} `json:"podcast"`
}

type PocketCastsNewReleasesResponse struct {
	Episodes []struct {
		UUID        string    `json:"uuid"`
		Title       string    `json:"title"`
		URL         string    `json:"url"`
		Podcast     string    `json:"podcastTitle"`
		PodcastUUID string    `json:"podcastUuid"`
		Date        time.Time `json:"published"`
		Duration    int       `json:"duration"`
		PlayedUpTo  int       `json:"playedUpTo"`
	}
}

func PocketCastsRequest(endpoint string, body *map[string]any, response any) error {
	URL := "https://"
	method := "POST"
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if body == nil {
		method = "GET"
	}
	if strings.Contains(endpoint, "pocketcasts.com") {
		URL += endpoint
	} else {
		URL += "api.pocketcasts.com" + endpoint
	}
	if endpoint != "/user/login" {
		if err := getToken(); err != nil {
			return fmt.Errorf("pocketcasts token not granted")
		}
		headers["Authorization"] = "Bearer " + pocketCastsToken
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	var req *http.Request
	var err error
	if body == nil {
		req, err = http.NewRequest(method, URL, nil)
	} else {
		if jsonBody, error := json.Marshal(body); error == nil {
			req, err = http.NewRequest(method, URL, bytes.NewBuffer(jsonBody))
		} else {
			return fmt.Errorf("error marshaling request body: %v", error)
		}
	}
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		os.Remove(".token")
		return PocketCastsRequest(endpoint, body, response)
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}
	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return fmt.Errorf("error decoding response: %v", err)
		}
	}
	return nil
}

func PocketCastsLogin(email, password string) error {
	body := map[string]any{
		"email":    email,
		"password": password,
	}
	var response struct {
		Token string `json:"token"`
	}
	if err := PocketCastsRequest("/user/login", &body, &response); err != nil {
		return err
	}
	pocketCastsToken = response.Token
	err := os.WriteFile(".token", []byte(response.Token), 0600)
	if err != nil {
		return fmt.Errorf("failed to store token: %v", err)
	}
	return nil
}

func GetPodcastList(force bool) error {
	if !force && len(podcastMap) > 0 {
		return nil
	}
	maxAge := 24 * time.Hour
	if force {
		maxAge = 0
	}
	file := getCachePath("podcast_list")

	podcastMap = make(map[string]*Podcast)
	if data, err := readCache(file, maxAge, "allPodcasts"); err == nil {
		if err := json.Unmarshal(data, &podcastMap); err == nil {
			return nil
		}
	}
	body := map[string]any{
		"v": 1,
	}
	var response PocketCastsPodcastsResponse
	if err := PocketCastsRequest("/user/podcast/list", &body, &response); err != nil {
		return err
	}
	podcastMap = make(map[string]*Podcast)
	for _, p := range response.Podcasts {
		_p := &Podcast{
			Name:        p.Name,
			Link:        p.Link,
			UUID:        p.UUID,
			Author:      p.Author,
			Desc:        p.Desc,
			LastUpdated: p.LastUpdated,
			Image:       fmt.Sprintf("https://static.pocketcasts.com/discover/images/webp/200/%s.webp", p.UUID),
		}
		podcastMap[p.UUID] = _p
	}
	data, _ := json.Marshal(podcastMap)
	writeCache(file, data)
	return nil
}

func GetUpNext(force bool) ([]*Episode, error) {
	episodes := make([]*Episode, 0)
	maxAge := 30 * time.Minute
	if force {
		maxAge = 0
	}
	file := getCachePath("up_next")

	if data, err := readCache(file, maxAge, "up_next"); err == nil {
		if err := json.Unmarshal(data, &episodes); err == nil {
			upNextMap = make(map[string]*Episode)
			for _, e := range episodes {
				upNextMap[e.UUID] = e
			}
			return episodes, nil
		}
	}
	if err := GetPodcastList(force); err != nil {
		return nil, err
	}
	body := map[string]any{
		"version":        2,
		"model":          "webplayer",
		"showPlayStatus": true,
	}
	var response PocketCastsUpNextResponse
	if err := PocketCastsRequest("/up_next/list", &body, &response); err != nil {
		return nil, err
	}

	return processUpNextResponse(&response)
}

func processUpNextResponse(response *PocketCastsUpNextResponse) ([]*Episode, error) {
	upNextMap = make(map[string]*Episode)
	if len(podcastMap) == 0 {
		GetPodcastList(false)
	}
	episodes := make([]*Episode, len(response.Episodes))

	for i, e := range response.Episodes {
		p, ok := podcastMap[e.PodcastUUID]
		if !ok {
			p = &Podcast{
				UUID: e.PodcastUUID,
			}
			podcastMap[e.PodcastUUID] = p
		}
		if p.Name == "" {
			p.GetInfo()
		}
		_e := &Episode{
			UUID:        e.UUID,
			Title:       e.Title,
			URL:         e.URL,
			Podcast:     p.Name,
			PodcastUUID: p.UUID,
			Date:        e.Date,
			Image:       fmt.Sprintf("https://static.pocketcasts.com/discover/images/webp/200/%s.webp", e.PodcastUUID),
		}
		upNextMap[e.UUID] = _e
		episodes[i] = _e
		if p.EpisodeMap == nil {
			p.EpisodeMap = make(map[string]*Episode)
		}
		p.EpisodeMap[e.UUID] = _e
	}

	for _, e := range response.EpisodeSync {
		if episode, ok := upNextMap[e.UUID]; ok {
			episode.PlayedUpTo = e.PlayedUpTo
			episode.Duration = e.Duration
		}
	}

	data, _ := json.Marshal(episodes)
	file := getCachePath("up_next")
	writeCache(file, data)

	return episodes, nil
}

func GetList(list string, force bool) ([]*Episode, error) {
	if list != "new_releases" && list != "history" {
		return nil, fmt.Errorf("invalid list: %s", list)
	}
	episodes := make([]*Episode, 0)
	maxAge := 12 * time.Hour
	if force {
		maxAge = 0
	}
	file := getCachePath(list)

	if data, err := readCache(file, maxAge, list); err == nil {
		if err := json.Unmarshal(data, &episodes); err == nil {
			return episodes, nil
		}
	}
	if err := GetPodcastList(force); err != nil {
		return nil, err
	}
	body := map[string]any{}
	var response PocketCastsNewReleasesResponse
	if err := PocketCastsRequest("/user/"+list, &body, &response); err != nil {
		return nil, err
	}
	for _, e := range response.Episodes {
		p, ok := podcastMap[e.PodcastUUID]
		if !ok {
			p = &Podcast{
				UUID: e.PodcastUUID,
				Name: e.Podcast,
			}
			podcastMap[e.PodcastUUID] = p
		}
		if p.Name == "" {
			p.GetInfo()
		}
		_e := &Episode{
			UUID:        e.UUID,
			Title:       e.Title,
			URL:         e.URL,
			Podcast:     p.Name,
			PodcastUUID: p.UUID,
			Date:        e.Date,
			Image:       fmt.Sprintf("https://static.pocketcasts.com/discover/images/webp/200/%s.webp", e.PodcastUUID),
		}
		episodes = append(episodes, _e)
		if p.EpisodeMap == nil {
			p.EpisodeMap = make(map[string]*Episode)
		}
		p.EpisodeMap[e.UUID] = _e
	}
	data, _ := json.Marshal(episodes)
	writeCache(file, data)
	return episodes, nil
}

func (p *Podcast) GetInfo() error {
	if p.UUID == "" {
		return fmt.Errorf("podcast UUID not set")
	}
	file := getCachePath("podcasts", p.UUID)
	if data, err := readCache(file, 0, "podcast", p.UUID); err == nil {
		if err := json.Unmarshal(data, &p); err == nil {
			return nil
		}
	}
	var response PocketCastsEpisodesResponse
	url := fmt.Sprintf("podcast-api.pocketcasts.com/podcast/full/%s", p.UUID)
	if err := PocketCastsRequest(url, nil, &response); err != nil {
		return err
	}
	p.Name = response.Podcast.Name
	p.Author = response.Podcast.Author
	p.Desc = response.Podcast.Desc
	p.Link = response.Podcast.Link
	p.Image = fmt.Sprintf("https://static.pocketcasts.com/discover/images/webp/200/%s.webp", p.UUID)
	data, _ := json.Marshal(p)
	writeCache(file, data)
	return nil
}

func (p *Podcast) GetEpisodes(force bool) error {
	if err := p.resolveMetadata(); err != nil {
		return err
	}
	maxAge := 12 * time.Hour
	if force {
		maxAge = 0
	}
	file := getCachePath("podcasts", p.UUID)
	if data, err := readCache(file, maxAge, "podcast", p.UUID); err == nil {
		if err := json.Unmarshal(data, &p); err == nil {
			return nil
		}
	}
	return p.fetchAndUpdateEpisodes()
}

func (p *Podcast) resolveMetadata() error {
	if p.UUID == "" {
		if p.Name != "" {
			if err := GetPodcastList(false); err == nil {
				for _, _p := range podcastMap {
					if _p.Name == p.Name {
						p.UUID = _p.UUID
						return nil
					}
				}
			}
		}
		return fmt.Errorf("podcast UUID not set")
	}
	return nil
}

func (p *Podcast) fetchAndUpdateEpisodes() error {
	type requestResult struct {
		response *PocketCastsEpisodesResponse
		err      error
	}
	ch1 := make(chan requestResult)
	ch2 := make(chan requestResult)

	go func() {
		var response PocketCastsEpisodesResponse
		url := fmt.Sprintf("podcast-api.pocketcasts.com/podcast/full/%s", p.UUID)
		err := PocketCastsRequest(url, nil, &response)
		ch1 <- requestResult{&response, err}
	}()

	go func() {
		var response PocketCastsEpisodesResponse
		url := fmt.Sprintf("podcast-api.pocketcasts.com/mobile/show_notes/full/%s", p.UUID)
		err := PocketCastsRequest(url, nil, &response)
		ch2 <- requestResult{&response, err}
	}()

	result1 := <-ch1
	result2 := <-ch2

	if result1.err != nil {
		return result1.err
	}
	if result2.err != nil {
		return result2.err
	}

	p.Name = result1.response.Podcast.Name
	p.Author = result1.response.Podcast.Author
	p.Desc = result1.response.Podcast.Desc
	p.Link = result1.response.Podcast.Link
	p.Image = fmt.Sprintf("https://static.pocketcasts.com/discover/images/webp/200/%s.webp", p.UUID)
	p.EpisodeMap = make(map[string]*Episode)

	for _, e := range result1.response.Podcast.Episodes {
		_e := &Episode{
			UUID:        e.UUID,
			Title:       e.Title,
			URL:         e.URL,
			Podcast:     p.Name,
			PodcastUUID: p.UUID,
			Date:        e.Date,
			Duration:    e.Duration,
			Image:       p.Image,
		}
		p.EpisodeMap[e.UUID] = _e
		if e.Date.After(p.LastUpdated) {
			p.LastUpdated = e.Date
		}
	}

	for _, e := range result2.response.Podcast.Episodes {
		if _e, ok := p.EpisodeMap[e.UUID]; ok {
			_e.ShowNotes = e.ShowNotes
			if e.Image != "" {
				_e.Image = e.Image
			}
		}
	}

	data, _ := json.Marshal(p)
	file := getCachePath("podcasts", p.UUID)
	writeCache(file, data)
	return nil
}
