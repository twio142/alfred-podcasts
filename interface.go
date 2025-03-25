package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

func ListPodcasts() {
	if err := GetAllPodcasts(false); err != nil {
		workflow.WarnEmpty(err.Error())
		return
	}
	if len(podcastMap) == 0 {
		item := Item{
			Title:    "No Podcasts Found",
			Subtitle: "Refresh",
			Icon:     &Icon{Path: "icons/refresh.png"},
		}
		item.SetVar("refresh", "allPodcasts")
		workflow.AddItem(&item)
		return
	}
	for _, p := range podcastMap {
		workflow.AddItem(p.Format())
	}
	sort.Slice(workflow.Items, func(i, j int) bool {
		_i := workflow.Items[i].UID
		_j := workflow.Items[j].UID
		return podcastMap[_i].LastUpdated.After(podcastMap[_j].LastUpdated)
	})
}

func ListNewReleases() {
	episodes, err := GetList("new_releases", false)
	if err != nil {
		workflow.WarnEmpty(err.Error())
		return
	}
	if len(episodes) == 0 {
		item := Item{
			Title:    "No Episodes Found",
			Subtitle: "Refresh all podcasts",
			Icon:     &Icon{Path: "icons/refresh.png"},
		}
		item.SetVar("refresh", "allPodcasts")
		workflow.AddItem(&item)
		return
	}
	for _, e := range episodes {
		if e.Duration == 0 {
			p := &Podcast{Name: e.Podcast, UUID: e.PodcastUUID}
			if err := p.GetEpisodes(false); err == nil {
				if _e, ok := p.EpisodeMap[e.UUID]; ok {
					e.Duration = _e.Duration
					e.ShowNotes = _e.ShowNotes
					e.Date = _e.Date
				}
			}
		}
		item := e.Format()
		alt := &Mod{Subtitle: "Refresh latest episodes", Icon: &Icon{Path: "icons/refresh.png"}}
		alt.SetVar("refresh", "new_release")
		item.Mods.Alt = alt
		workflow.AddItem(item)
	}
}

func (p *Podcast) ListEpisodes() {
	if p == nil {
		workflow.WarnEmpty("Podcast Not Found")
		return
	} else if len(p.EpisodeMap) == 0 {
		item := Item{
			Title:    "No Episodes Found",
			Subtitle: "Refresh podcast",
			Icon:     &Icon{Path: "icons/refresh.png"},
		}
		item.SetVar("refresh", "podcast")
		workflow.AddItem(&item)
		return
	}
	episodes := make([]*Episode, 0, len(p.EpisodeMap))
	for _, e := range p.EpisodeMap {
		episodes = append(episodes, e)
	}
	sort.Slice(episodes, func(i, j int) bool {
		return episodes[i].Date.After(episodes[j].Date)
	})
	for i, e := range episodes {
		if i == 30 {
			break
		}
		item := e.Format()
		item.Subtitle = fmt.Sprintf("􀪔 %s  ·  %s", e.Podcast, item.Subtitle)
		item.Match = matchString(e.Title)
		item.AutoComplete = ""
		alt := &Mod{Subtitle: "Refresh podcast", Icon: &Icon{Path: "icons/refresh.png"}}
		alt.SetVar("refresh", "podcast")
		alt.SetVar("podcast", e.Podcast)
		item.Mods.Alt = alt
		item.Mods.Shift = nil
		workflow.AddItem(item)
	}
	item := Item{
		Title: "Go Back",
		Icon:  &Icon{Path: "icons/back.png"},
	}
	item.SetVar("trigger", "podcasts")
	workflow.AddItem(&item)
}

func ListUpNext() {
	episodes, err := GetUpNext(false)
	if err != nil {
		workflow.WarnEmpty(err.Error())
		return
	}
	if len(episodes) == 0 {
		item := Item{
			Title:    "No Episodes Found",
			Subtitle: "Refresh all podcasts",
			Icon:     &Icon{Path: "icons/refresh.png"},
		}
		item.SetVar("refresh", "allPodcasts")
		workflow.AddItem(&item)
		return
	}
	for _, e := range episodes {
		if e.Duration == 0 {
			p := &Podcast{Name: e.Podcast, UUID: e.PodcastUUID}
			if err := p.GetEpisodes(false); err == nil {
				if _e, ok := p.EpisodeMap[e.UUID]; ok {
					e.Duration = _e.Duration
					e.ShowNotes = _e.ShowNotes
					e.Date = _e.Date
				}
			}
		}
		item := e.Format()
		alt := &Mod{Subtitle: "Refresh queue", Icon: &Icon{Path: "icons/refresh.png"}}
		alt.SetVar("refresh", "up_next")
		item.Mods.Alt = alt
		workflow.AddItem(item)
	}
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
		item := e.Format()
		valid := false
		item.Valid = &valid
		item.Mods.Cmd = nil
		workflow.AddItem(item)
	} else {
		workflow.WarnEmpty("No Episode Playing")
	}
}

func (p *Podcast) Format() *Item {
	icon := getCachePath("artworks", p.UUID)
	_, err := os.Stat(icon)
	if err != nil {
		icon = ""
	}
	item := Item{
		Title:        p.Name,
		Subtitle:     p.Desc,
		Match:        matchString(p.Name),
		UID:          p.UUID,
		QuickLookURL: p.Link,
		Text: struct {
			Copy      string `json:"copy,omitempty"`
			LargeType string `json:"largetype,omitempty"`
		}{LargeType: p.Desc},
		Icon: &Icon{Path: icon},
		Mods: struct {
			Cmd      *Mod `json:"cmd,omitempty"`
			Alt      *Mod `json:"alt,omitempty"`
			Shift    *Mod `json:"shift,omitempty"`
			Ctrl     *Mod `json:"ctrl,omitempty"`
			AltShift *Mod `json:"alt+shift,omitempty"`
		}{},
	}
	item.SetVar("trigger", "episodes")
	item.SetVar("podcast", p.Name)
	item.SetVar("podcastUuid", p.UUID)

	alt := &Mod{Subtitle: "Refresh podcast", Icon: &Icon{Path: "icons/refresh.png"}}
	alt.SetVar("refresh", "podcasts/"+p.Name)
	item.Mods.Alt = alt

	altShift := &Mod{Subtitle: "Refresh all podcasts", Icon: &Icon{Path: "icons/refresh.png"}}
	altShift.SetVar("refresh", "allPodcasts")
	item.Mods.AltShift = altShift

	ctrl := &Mod{Subtitle: "Unsubscribe", Icon: &Icon{Path: "icons/trash.png"}, Arg: p.URL}
	ctrl.SetVar("action", "unsubscribe")
	item.Mods.Ctrl = ctrl
	return &item
}

func (e *Episode) Format() *Item {
	icon := getCachePath("artworks", e.PodcastUUID)
	if _, err := os.Stat(icon); err != nil {
		icon = ""
	}
	item := Item{
		Title:        e.Title,
		Subtitle:     fmt.Sprintf("􀉉 %s  ·  􀖈 %s", e.Date.Format("Mon, 2006-01-02"), formatDuration(e.Duration)),
		Arg:          e.URL,
		UID:          e.UUID,
		QuickLookURL: e.CacheShownote(),
		Icon:         &Icon{Path: icon},
		Match:        matchString(e.Title, e.Podcast),
		AutoComplete: e.Podcast,
		Mods: struct {
			Cmd      *Mod `json:"cmd,omitempty"`
			Alt      *Mod `json:"alt,omitempty"`
			Shift    *Mod `json:"shift,omitempty"`
			Ctrl     *Mod `json:"ctrl,omitempty"`
			AltShift *Mod `json:"alt+shift,omitempty"`
		}{},
	}
	item.SetVar("actionKeep", "addToQueue")
	item.SetVar("podcast", e.Podcast)
	item.SetVar("url", e.URL)
	item.SetVar("uuid", e.UUID)
	item.SetVar("podcastUuid", e.PodcastUUID)

	cmd := &Mod{Subtitle: "Play now", Icon: &Icon{Path: "icons/play.png"}}
	cmd.SetVar("actionKeep", "play_now")
	cmd.SetVar("url", e.URL)
	item.Mods.Cmd = cmd

	shift := &Mod{Subtitle: "􀪔 " + e.Podcast}
	shift.SetVar("trigger", "episodes")
	shift.SetVar("podcast", e.Podcast)
	item.Mods.Shift = shift
	return &item
}

func generatePlaylist() (string, error) {
	episodes, err := GetUpNext(false)
	if err != nil {
		return "", err
	}
	list := make([]string, 0, len(episodes)*2)
	for _, e := range episodes {
		list = append(list, fmt.Sprintf("%s —— %s", e.Podcast, e.Title))
		list = append(list, e.URL)
	}
	file := fmt.Sprintf("Podcasts (%dx) %s.m3u", len(episodes), time.Now().Format("2006-01-02 15.04.05"))
	file = getCachePath(file)
	if err := writeCache(file, []byte(strings.Join(list, "\n"))); err != nil {
		return "", err
	}
	return file, nil
}
