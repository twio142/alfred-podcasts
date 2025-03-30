package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/semaphore"
)

type Podcast struct {
	Name        string              `json:"name"`
	Author      string              `json:"author"`
	URL         string              `json:"-"`
	Desc        string              `json:"desc"`
	Image       string              `json:"image"`
	Link        string              `json:"link"`
	EpisodeMap  map[string]*Episode `json:"episodes,omitempty"`
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
	Played      bool      `json:"-"`
	Image       string    `json:"image"`
	UUID        string    `json:"uuid"`
}

func GetAllPodcasts(force bool) error {
	if err := GetPodcastList(force); err != nil {
		return err
	}

	sem := semaphore.NewWeighted(50)
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
	file := getCachePath("artworks", p.UUID)
	if _, err := os.Stat(file); os.IsNotExist(err) && p.Image != "" {
		downloadImage(p.Image, file)
	}
}

func (p *Podcast) ClearCache() {
	if p.UUID == "" {
		return
	}
	scpt := fmt.Sprintf(`find "%s" -type f -name "%s*" -delete`, cacheDir, p.UUID)
	cmd := exec.Command("/bin/sh", "-c", scpt)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	cmd.Start()
}

func (e *Episode) CacheShownotes() string {
	file := getCachePath("shownotes", fmt.Sprintf("%s.%s.md", e.PodcastUUID, e.UUID))
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

func GetPlaying() {
	title := os.Getenv("title")
	author := os.Getenv("author")
	podcast := os.Getenv("podcast")
	if title == "" {
		cmd := exec.Command("nowplaying-cli", "get", "title", "artist", "album")
		if out, err := cmd.Output(); err == nil {
			output := strings.Split(string(out), "\n")
			title = output[0]
			author = output[1]
			podcast = output[1]
		}
	}
	if e := FindEpisode(map[string]string{"title": title, "podcast": podcast, "author": author}); e != nil {
		item := e.Format(false)
		valid := false
		item.Valid = &valid
		item.Mods.Cmd = nil
		workflow.AddItem(item)
	} else {
		workflow.WarnEmpty("No Episode Playing")
	}
}

func ExportPlaylist() (string, error) {
	episodes, err := GetUpNext(false)
	if err != nil {
		return "", err
	}
	list := make([]string, 0, len(episodes)*2)
	for _, e := range episodes {
		list = append(list, fmt.Sprintf("# %s\t%s", e.Podcast, e.Title))
		u := e.URL
		if e.PlayedUpTo > 0 {
			// NOTE: add timestamp to URL
			// requires custom configs in mpv (event handling)
			parsedURL, err := url.Parse(u)
			if err != nil {
				return "", fmt.Errorf("failed to parse URL %s: %v", u, err)
			}
			q := parsedURL.Query()
			q.Set("t", fmt.Sprintf("%d", e.PlayedUpTo))
			parsedURL.RawQuery = q.Encode()
			u = parsedURL.String()
		}
		list = append(list, u)
	}
	file := "podcast_playlist.m3u"
	file = getCachePath(file)
	if err := writeCache(file, []byte(strings.Join(list, "\n"))); err != nil {
		return "", err
	}
	return file, nil
}

func SyncPlaylist() error {
	episodes, err := getPlaybackState()
	if err != nil {
		return err
	}
	var errs []error
	for _, e := range episodes {
		if e.Played {
			if err := e.Archive(true); err != nil {
				errs = append(errs, err)
			}
		} else if e.PlayedUpTo > 0 {
			if err := e.Update(map[string]any{
				"position": fmt.Sprintf("%d", e.PlayedUpTo),
				"status": 2,
			}); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("sync playlist errors: %v", errs)
	}
	return nil
}
