package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

type Podcast struct {
	Name        string    `json:"name"`
	Author      string    `json:"author"`
	URL         string    `json:"url"`
	Desc        string    `json:"desc"`
	Image       string    `json:"image"`
	Link        string    `json:"link"`
	Episodes    []Episode `json:"episodes"`
	LastUpdated time.Time `json:"lastUpdated"`
}

type Episode struct {
	Title    string    `json:"title"`
	URL      string    `json:"url"`
	Html     string    `json:"html"`
	Podcast  string    `json:"podcast"`
	Date     time.Time `json:"date"`
	Duration int       `json:"duration"`
	Image    string    `json:"image"`
}

func GetAllPodcasts(force bool) error {
	defer func() {
		sort.Slice(allPodcasts, func(i, j int) bool {
			return allPodcasts[i].LastUpdated.After(allPodcasts[j].LastUpdated)
		})
	}()

	sem := semaphore.NewWeighted(50)

	if !force {
		if len(allPodcasts) > 0 {
			return nil
		}
		if files, err := os.ReadDir(getCachePath("podcasts")); err == nil && len(files) > 0 {
			var podcasts []*Podcast
			podcastCh := make(chan *Podcast, len(files))
			var wg sync.WaitGroup
			for _, file := range files {
				if file.Name() == ".DS_Store" || file.IsDir() {
					continue
				}
				decodedName, _ := url.PathUnescape(file.Name())
				wg.Add(1)
				go func(p *Podcast) {
					if err := sem.Acquire(context.Background(), 1); err != nil {
						wg.Done()
						return
					}
					defer sem.Release(1)
					defer wg.Done()
					p.GetEpisodes(false)
					podcastCh <- p
				}(&Podcast{Name: decodedName})
			}
			go func() {
				wg.Wait()
				close(podcastCh)
			}()
			for p := range podcastCh {
				podcasts = append(podcasts, p)
			}
			if len(podcasts) > 0 {
				allPodcasts = podcasts
				return nil
			}
		}
	}

	podcasts, err := RequestFeeds()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	for _, p := range podcasts {
		wg.Add(1)
		go func(p *Podcast) {
			if err := sem.Acquire(context.Background(), 1); err != nil {
				wg.Done()
				return
			}
			defer sem.Release(1)
			defer wg.Done()
			p.GetEpisodes(force)
			p.CacheArtwork()
		}(p)
	}
	wg.Wait()
	allPodcasts = podcasts
	return nil
}

func GetLatestEpisodes(force bool) []*Episode {
	var episodes []*Episode
	var maxAge = time.Duration(math.MaxInt64)
	if force {
		maxAge = 0
	}
	path := getCachePath("latest")

	if data, err := readCache(path, maxAge); err == nil {
		json.Unmarshal(data, &episodes)
		return episodes
	} else {
		var days = 30
		if d, err := strconv.Atoi(os.Getenv("PODCAST_CACHE_DAYS")); err == nil {
			days = d
		}
		GetAllPodcasts(force)
		for _, p := range allPodcasts {
			for _, e := range p.Episodes {
				if e.Date.After(time.Now().AddDate(0, 0, -days)) {
					episodes = append(episodes, &e)
				}
			}
		}

		if len(episodes) == 0 {
			return nil
		}
		sort.Slice(episodes, func(i, j int) bool {
			return episodes[i].Date.After(episodes[j].Date)
		})
		data, _ = json.Marshal(episodes)
		writeCache(path, data)
		return episodes
	}
}

func refreshLatest() {
	var latestEpisodes []*Episode
	var episodesToKeep []*Episode
	path := getCachePath("latest")

	if data, err := readCache(path, time.Duration(math.MaxInt64)); err == nil {
		json.Unmarshal(data, &latestEpisodes)
	} else {
		os.Remove(getCachePath("latest"))
		return
	}

	items, err := GetPlaylist()
	if err != nil || len(items) == 0 {
		os.Remove(getCachePath("latest"))
		return
	}
	for i, item := range items {
		if item.Current {
			items = items[i:]
			break
		}
	}
	for _, item := range items {
		for _, e := range latestEpisodes {
			if e.URL == item.Filename {
				episodesToKeep = append(episodesToKeep, e)
				break
			}
		}
	}
	os.Remove(getCachePath("latest"))
	if len(episodesToKeep) == 0 {
		return
	}

	latestEpisodes = GetLatestEpisodes(false)
	count := len(latestEpisodes)
	threshold := latestEpisodes[len(latestEpisodes)-1].Date
	for _, e := range episodesToKeep {
		if e.Date.Before(threshold) {
			latestEpisodes = append(latestEpisodes, e)
		}
	}
	if len(latestEpisodes) > count {
		data, _ := json.Marshal(latestEpisodes)
		writeCache(path, data)
	}
}

func AddToLatest(url string, name string) {
	if url == "" || name == "" {
		return
	}
	latestEpisodes := GetLatestEpisodes(false)
	for _, episode := range latestEpisodes {
		if episode.URL == url && episode.Podcast == name {
			return
		}
	}
	if episode := FindEpisode(map[string]string{"url": url, "podcast": name}); episode != nil {
		latestEpisodes = append(latestEpisodes, episode)
		data, _ := json.Marshal(latestEpisodes)
		writeCache(getCachePath("latest"), data)
	}
}

func FindEpisode(args map[string]string) *Episode {
	url := args["url"]
	title := args["title"]
	podcast := args["podcast"]
	author := args["author"]
	if url == "" && title == "" {
		return nil
	}
	if podcast != "" {
		p := &Podcast{Name: podcast}
		p.GetEpisodes(false)
		for _, e := range p.Episodes {
			if (url != "" && e.URL == url) || (title != "" && e.Title == title) {
				return &e
			}
		}
	}
	if author != "" {
		GetAllPodcasts(false)
		for _, p := range allPodcasts {
			if p.Author == author {
				for _, e := range p.Episodes {
					if (url != "" && e.URL == url) || (title != "" && e.Title == title) {
						return &e
					}
				}
				break
			}
		}
	}
	for _, e := range GetLatestEpisodes(false) {
		if (url != "" && e.URL == url) || (title != "" && e.Title == title) {
			return e
		}
	}
	return nil
}

func (p *Podcast) CacheArtwork() {
	path := getCachePath("artworks", p.Name)
	if _, err := os.Stat(path); os.IsNotExist(err) && p.Image != "" {
		downloadImage(p.Image, path)
	}
}

func (p *Podcast) GetEpisodes(force bool) error {
	var maxAge = 24 * time.Hour
	if force {
		maxAge = 0
	} else if len(p.Episodes) > 0 {
		return nil
	}
	path := getCachePath("podcasts", p.Name)

	if data, err := readCache(path, maxAge); err == nil {
		json.Unmarshal(data, &p)
	} else {
		if p.URL == "" {
			feeds, err := RequestFeeds()
			if err != nil {
				return err
			}
			for _, feed := range feeds {
				if feed.Name == p.Name {
					p.URL = feed.URL
					break
				}
			}
			if p.URL == "" {
				return fmt.Errorf("podcast not found")
			}
		}
		rss, err := RequestRss(p.URL)
		if err != nil {
			return err
		}
		if p.Name == "" {
			p.Name = strings.TrimSpace(rss.Channel.Title)
			path = getCachePath("podcasts", p.Name)
		}
		p.Desc = rss.desc()
		p.Image = longestString(rss.Channel.Image.Href, rss.Channel.Image.URL)
		p.Link = rss.Channel.Link
		p.Author = rss.Channel.Author
		for _, item := range rss.Channel.Items {
			e := Episode{
				Title:    strings.TrimSpace(strings.ReplaceAll(item.Title, "&amp;", "&")),
				Html:     longestString(item.Desc, item.Content, item.Summary),
				Date:     parseDate(item.Date),
				Podcast:  p.Name,
				Duration: calculateDuration(item.Duration),
				Image:    longestString(item.Image.Href, item.Image.URL, p.Image),
			}
			e.URL = item.Enclosure.URL
			if e.URL == "" {
				e.URL = item.Link
			}
			p.Episodes = append(p.Episodes, e)
			if e.Date.After(p.LastUpdated) {
				p.LastUpdated = e.Date
			}
		}
		sort.Slice(p.Episodes, func(i, j int) bool {
			return p.Episodes[i].Date.After(p.Episodes[j].Date)
		})
		data, _ = json.Marshal(p)
		writeCache(path, data)
	}
	return nil
}

func (p *Podcast) ClearCache() {
	os.Remove(getCachePath("podcasts", p.Name))
	os.Remove(getCachePath("artworks", p.Name))
}

func (e *Episode) CacheShownote() string {
	path := getCachePath("shownotes", fmt.Sprintf("%s_%s.md", e.Podcast, e.Title))
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if e.Html == "" {
		return ""
	}
	re := regexp.MustCompile(`(<(p|span) [^>]*style="[^"]*)background-color:.+?; ?`)
	var html string
	html = re.ReplaceAllString(e.Html, "$1")
	re = regexp.MustCompile(`(<(p|span) [^>]*style=("[^"]+[^-]|"))color:.+?; ?`)
	html = re.ReplaceAllString(html, "$1")
	re = regexp.MustCompile(`<audio[^>]*(>[\s\S]*?</audio|/)>`)
	html = re.ReplaceAllString(html, "")
	html = strings.ReplaceAll(html, "\n", "<br/>")
	if e.Image != "" {
		html += "\n\n<img width=\"20%\" src=\"" + e.Image + "\"/>"
	}
	os.WriteFile(path, []byte(html), 0644)
	return path
}
