package main

import (
	"time"
	"os"
	"net/url"
	"fmt"
	"sync"
	"sort"
	"math"
	"encoding/json"
	"strings"
	"regexp"
	"strconv"
	"golang.org/x/sync/semaphore"
	"context"
)

type Podcast struct {
	Name  string `json:"name"`
	URL   string  `json:"url"`
	Desc  string `json:"desc"`
	Image string `json:"image"`
	Link  string `json:"link"`
	Episodes []Episode `json:"episodes"`
	LastUpdated time.Time `json:"lastUpdated"`
}

type Episode struct {
	Title string `json:"title"`
	URL string `json:"url"`
	Html  string `json:"html"`
	Author string `json:"author"`
	Date time.Time `json:"date"`
	Duration int `json:"duration"`
	Image string `json:"image"`
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

func AddToLatest(url string, name string) {
	if url == "" || name == "" {
		return
	}
	latestEpisodes := GetLatestEpisodes(false)
	for _, episode := range latestEpisodes {
		if episode.URL == url && episode.Author == name {
			return
		}
	}
	if episode := FindEpisode(map[string]string{"url": url, "author": name}); episode != nil {
		latestEpisodes = append(latestEpisodes, episode)
		data, _ := json.Marshal(latestEpisodes)
		writeCache(getCachePath("latest"), data)
	}
}

func FindEpisode(args map[string]string) *Episode {
	url := args["url"]
	title := args["title"]
	author := args["author"]
	if url == "" && title == "" {
		return nil
	}
	if author != "" {
		p := &Podcast{Name: author}
		p.GetEpisodes(false)
		for _, e := range p.Episodes {
			if (url != "" && e.URL == url) || (title != "" && e.Title == title) {
				return &e
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
		p.Desc = rss.desc()
		p.Image = longestString(rss.Channel.Image.Href, rss.Channel.Image.URL)
		p.Link = rss.Channel.Link
		for _, item := range rss.Channel.Items {
			e := Episode{
				Title: strings.TrimSpace(strings.ReplaceAll(item.Title, "&amp;", "&")),
				Html: longestString(item.Desc, item.Content, item.Summary),
				Date: parseDate(item.Date),
				Author: p.Name,
				Duration: calculateDuration(item.Duration),
				Image: longestString(item.Image.Href, item.Image.URL, p.Image),
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

func (e *Episode) CacheShownote() string {
	path := getCachePath("shownotes", fmt.Sprintf("%s_%s.md", e.Author, e.Title))
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
	html = strings.ReplaceAll(html, "\n", "<br/>")
	if e.Image != "" {
		html += "\n\n<img width=\"20%\" src=\"" + e.Image + "\"/>"
	}
	os.WriteFile(path, []byte(html), 0644)
	return path
}
