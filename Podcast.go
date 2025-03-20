package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
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
	sem := semaphore.NewWeighted(50)

	if err := GetPodcasts(force); err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, p := range podcastMap {
		wg.Add(1)
		go func(p *Podcast) {
			defer wg.Done()
			if err := sem.Acquire(context.Background(), 1); err != nil {
				return
			}
			defer sem.Release(1)
			if err := p.GetEpisodes(force); err != nil {
				fmt.Fprintf(os.Stderr, "[%s]: %s\n", p.Name, err)
			}
			p.CacheArtwork()
		}(p)
	}
	wg.Wait()
	return nil
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
		for _, e := range p.EpisodeMap {
			if (url != "" && e.URL == url) || (title != "" && e.Title == title) {
				return e
			}
		}
	}
	if author != "" {
		GetAllPodcasts(false)
		for _, p := range podcastMap {
			if p.Author == author {
				for _, e := range p.EpisodeMap {
					if (url != "" && e.URL == url) || (title != "" && e.Title == title) {
						return e
					}
				}
			}
		}
	}

	episodes, _ := GetList("up_next", false)
	for _, e := range episodes {
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
