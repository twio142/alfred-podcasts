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
	Name        string              `json:"name"`
	Author      string              `json:"author"`
	URL         string              `json:"url"`
	Desc        string              `json:"desc"`
	Image       string              `json:"image"`
	Link        string              `json:"link"`
	Episodes    []Episode           `json:"-"`
	EpisodeMap  map[string]*Episode `json:"episodes"`
	LastUpdated time.Time           `json:"lastUpdated"`
	UUID        string              `json:"uuid"`
}

type Episode struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	ShowNotes   string    `json:"show_notes"`
	Podcast     string    `json:"podcast"`
	PodcastUUID string    `json:"podcast_uuid"`
	Date        time.Time `json:"date"`
	Duration    int       `json:"duration"`
	PlayedUpTo  int       `json:"playedUpTo"`
	Image       string    `json:"image"`
	UUID        string    `json:"uuid"`
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
	maxAge := time.Duration(math.MaxInt64)
	if force {
		maxAge = 0
	}
	file := getCachePath("latest")

	if data, err := readCache(file, maxAge); err == nil {
		if err := json.Unmarshal(data, &episodes); err == nil {
			return episodes
		}
	}
	days := 30
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
	data, _ := json.Marshal(episodes)
	writeCache(file, data)
	return episodes
}

func refreshLatest() {
	var latestEpisodes []*Episode
	var episodesToKeep []*Episode
	file := getCachePath("latest")

	if data, err := readCache(file, time.Duration(math.MaxInt64)); err != nil {
		os.Remove(file)
		return
	} else {
		json.Unmarshal(data, &latestEpisodes)
	}

	items, err := GetPlaylist()
	if err != nil || len(items) == 0 {
		os.Remove(file)
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
	os.Remove(file)
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
		writeCache(file, data)
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
	file := getCachePath("artworks", p.Name)
	if _, err := os.Stat(file); os.IsNotExist(err) && p.Image != "" {
		downloadImage(p.Image, file)
	}
}

func (p *Podcast) GetEpisodes(force bool) error {
	maxAge := 12 * time.Hour
	if force {
		maxAge = 0
	} else if len(p.Episodes) > 0 {
		return nil
	}
	file := getCachePath("podcasts", p.Name)
	if data, err := readCache(file, maxAge); err == nil {
		if err := json.Unmarshal(data, &p); err == nil {
			return nil
		}
	}
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
		file = getCachePath("podcasts", p.Name)
	}
	p.Desc = rss.desc()
	p.Image = longestString(rss.Channel.Image.Href, rss.Channel.Image.URL)
	p.Link = rss.Channel.Link
	p.Author = rss.Channel.Author
	for _, item := range rss.Channel.Items {
		e := Episode{
			Title:     strings.TrimSpace(strings.ReplaceAll(item.Title, "&amp;", "&")),
			ShowNotes: longestString(item.Desc, item.Content, item.Summary),
			Date:      parseDate(item.Date),
			Podcast:   p.Name,
			Duration:  calculateDuration(item.Duration),
			Image:     longestString(item.Image.Href, item.Image.URL, p.Image),
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
	data, _ := json.Marshal(p)
	writeCache(file, data)
	return nil
}

func (p *Podcast) ClearCache() {
	os.Remove(getCachePath("podcasts", p.Name))
	os.Remove(getCachePath("artworks", p.Name))
}

func (e *Episode) CacheShownote() string {
	file := getCachePath("shownotes", fmt.Sprintf("%s_%s.md", e.Podcast, e.Title))
	if _, err := os.Stat(file); err == nil {
		return file
	}
	if e.ShowNotes == "" {
		return ""
	}
	re := regexp.MustCompile(`(<(p|span) [^>]*style="[^"]*)background-color:.+?; ?`)
	var showNotes string
	showNotes = re.ReplaceAllString(e.ShowNotes, "$1")
	re = regexp.MustCompile(`(<(p|span) [^>]*style=("[^"]+[^-]|"))color:.+?; ?`)
	showNotes = re.ReplaceAllString(showNotes, "$1")
	re = regexp.MustCompile(`<audio[^>]*(>[\s\S]*?</audio|/)>`)
	showNotes = re.ReplaceAllString(showNotes, "")
	showNotes = strings.ReplaceAll(showNotes, "\n", "<br/>")
	if e.Image != "" {
		showNotes += "\n\n<img width=\"20%\" src=\"" + e.Image + "\"/>"
	}
	os.WriteFile(file, []byte(showNotes), 0644)
	return file
}
