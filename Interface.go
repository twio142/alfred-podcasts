package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func ListPodcasts() {
	GetAllPodcasts(false)
	if len(allPodcasts) == 0 {
		item := Item{
			Title:    "No Podcasts Found",
			Subtitle: "Refresh",
			Icon:     &Icon{Path: "icons/refresh.png"},
		}
		item.SetVar("action", "refreshAll")
		workflow.AddItem(&item)
		return
	}
	for _, p := range allPodcasts {
		workflow.AddItem(p.Format())
	}
}

func ListLatest() {
	episodes := GetLatestEpisodes(false)
	if len(episodes) == 0 {
		item := Item{
			Title:    "No Episodes Found",
			Subtitle: "Refresh all podcasts",
			Icon:     &Icon{Path: "icons/refresh.png"},
		}
		item.SetVar("action", "refreshAll")
		workflow.AddItem(&item)
		return
	}
	for _, e := range episodes {
		item := e.Format()
		alt := &Mod{Subtitle: "Refresh latest episodes", Icon: &Icon{Path: "icons/refresh.png"}}
		alt.SetVar("action", "refreshLatest")
		item.Mods.Alt = alt
		workflow.AddItem(item)
	}
}

func ListEpisodes() {
	name := os.Getenv("podcast")
	if podcast == nil {
		podcast = &Podcast{Name: name}
		podcast.GetEpisodes(false)
		workflow.SetVar("podcast", name)
	}
	if podcast == nil {
		workflow.WarnEmpty("Podcast Not Found")
		return
	} else if len(podcast.Episodes) == 0 {
		item := Item{
			Title:    "No Episodes Found",
			Subtitle: "Refresh podcast",
			Icon:     &Icon{Path: "icons/refresh.png"},
		}
		item.SetVar("action", "refresh")
		workflow.AddItem(&item)
		return
	}
	for i, e := range podcast.Episodes {
		if i > 30 {
			break
		}
		item := e.Format()
		item.Subtitle = fmt.Sprintf("􀪔 %s  ·  %s", e.Podcast, item.Subtitle)
		item.Match = matchString(e.Title)
		item.AutoComplete = ""
		alt := &Mod{Subtitle: "Refresh podcast", Icon: &Icon{Path: "icons/refresh.png"}}
		alt.SetVar("action", "refresh")
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

func showSavedPlaylist() error {
	fileInfo, err := os.Stat(fmt.Sprintf("%s/playlist.m3u", cacheDir))
	if err != nil {
		return errors.New("no saved playlist found")
	}
	days := int(time.Since(fileInfo.ModTime()).Hours() / 24)
	since := fmt.Sprintf("%d days ago", days)
	switch days {
	case 0:
		since = "today"
	case 1:
		since = "yesterday"
	}
	item := &Item{
		Title:    "No Episodes Found",
		Subtitle: fmt.Sprintf("Load saved playlist (%s)", since),
		Arg:      fmt.Sprintf("%s/playlist.m3u", cacheDir),
		Type:     "file",
		Icon:     &Icon{Path: "icons/save.png"},
	}
	item.SetVar("action", "loadList")
	workflow.AddItem(item)
	return nil
}

func ListQueue() error {
	playlist, err := GetPlaylist()
	if err != nil || len(playlist) == 0 {
		return errors.New("no episodes found")
	}
	var episodes []*Episode
	latestEpisodes := GetLatestEpisodes(false)
	for _, i := range playlist {
		if !i.Current && len(episodes) == 0 {
			continue
		}
		for _, e := range latestEpisodes {
			if e.URL == i.Filename {
				if i.Current && e.Duration == 0 {
					if duration, err := runCommand("get_property", "duration"); err == nil {
						e.Duration = int(duration.(float64))
					} else {
						e.Duration = 999
					}
				}
				item := e.Format()
				valid := false
				item.Valid = &valid
				if len(episodes) > 1 {
					alt := &Mod{Subtitle: "Play next", Icon: &Icon{Path: "icons/moveUp.png"}}
					alt.SetVar("action", "playNext")
					alt.SetVar("url", e.URL)
					item.Mods.Alt = alt
				} else if len(episodes) == 0 {
					item.Mods.Cmd = nil
				}
				ctrl := &Mod{Subtitle: "Remove from queue", Icon: &Icon{Path: "icons/trash.png"}}
				ctrl.SetVar("action", "remove")
				ctrl.SetVar("id", i.ID)
				item.Mods.Ctrl = ctrl
				workflow.AddItem(item)
				episodes = append(episodes, e)
				break
			}
		}
	}
	if len(episodes) == 0 {
		return errors.New("no episodes found")
	} else {
		workflow.UnshiftItem(PlayerControl(episodes))
		return nil
	}
}

func GetPlaying() {
	var title string
	var author string
	title = os.Getenv("title")
	author = os.Getenv("artist")
	if title == "" {
		cmd := exec.Command("nowplaying-cli", "get", "title", "artist")
		if out, err := cmd.Output(); err == nil {
			output := strings.Split(string(out), "\n")
			title = output[0]
			author = output[1]
		}
	}
	if e := FindEpisode(map[string]string{"title": title, "author": author}); e != nil {
		item := e.Format()
		valid := false
		item.Valid = &valid
		item.Mods.Cmd = nil
		workflow.AddItem(item)
	} else if err := showSavedPlaylist(); err != nil {
		workflow.WarnEmpty("No Episode Playing")
	}
}

func (p *Podcast) Format() *Item {
	icon := getCachePath("artworks", p.Name)
	_, err := os.Stat(icon)
	if err != nil {
		icon = ""
	}
	item := Item{
		Title:        p.Name,
		Subtitle:     p.Desc,
		Match:        matchString(p.Name),
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

	alt := &Mod{Subtitle: "Refresh podcast", Icon: &Icon{Path: "icons/refresh.png"}}
	alt.SetVar("action", "refresh")
	alt.SetVar("podcast", p.Name)
	item.Mods.Alt = alt

	altShift := &Mod{Subtitle: "Refresh all podcasts", Icon: &Icon{Path: "icons/refresh.png"}}
	altShift.SetVar("action", "refreshAll")
	item.Mods.AltShift = altShift

	ctrl := &Mod{Subtitle: "Unsubscribe", Icon: &Icon{Path: "icons/trash.png"}, Arg: p.URL}
	ctrl.SetVar("action", "unsubscribe")
	item.Mods.Ctrl = ctrl
	return &item
}

func (e *Episode) Format() *Item {
	icon := getCachePath("artworks", e.Podcast)
	if _, err := os.Stat(icon); err != nil {
		icon = ""
	}
	item := Item{
		Title:        e.Title,
		Subtitle:     fmt.Sprintf("􀉉 %s  ·  􀖈 %s", e.Date.Format("Mon, 2006-01-02"), formatDuration(e.Duration)),
		Arg:          e.URL,
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

	cmd := &Mod{Subtitle: "Play now", Icon: &Icon{Path: "icons/play.png"}}
	cmd.SetVar("actionKeep", "play")
	cmd.SetVar("url", e.URL)
	item.Mods.Cmd = cmd

	shift := &Mod{Subtitle: "􀪔 " + e.Podcast}
	shift.SetVar("trigger", "episodes")
	shift.SetVar("podcast", e.Podcast)
	item.Mods.Shift = shift
	return &item
}
