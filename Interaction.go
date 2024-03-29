package main

import (
	"os"
	"fmt"
)

func ListPodcasts() {
	GetAllPodcasts(false)
	if len(allPodcasts) == 0 {
		item := Item {
			Title:    "No Podcasts Found",
			Subtitle: "Refresh",
			Valid:    true,
			Icon:     struct{ Path string `json:"path"` }{Path: "icons/refresh.png"},
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
			Valid:    true,
			Icon:     struct{ Path string `json:"path"` }{Path: "icons/refresh.png"},
		}
		item.SetVar("action", "refreshAll")
		workflow.AddItem(&item)
		return
	}
	for _, e := range episodes {
		item := e.Format()
		alt := Mod{Subtitle: "Refresh latest episodes", Valid: true, Icon: struct {
			Path string `json:"path"`
		}{Path: "icons/refresh.png"}}
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
			Valid:    true,
			Icon:     struct{ Path string `json:"path"` }{Path: "icons/refresh.png"},
		}
		item.SetVar("action", "refresh")
		workflow.AddItem(&item)
		return
	}
	for i, e := range podcast.Episodes {
		if i > 20 {
			break
		}
		item := e.Format()
		item.Subtitle = fmt.Sprintf("📻  %s  ·  %s", e.Author, item.Subtitle)
		item.Match = matchString(e.Title)
		item.AutoComplete = ""
		alt := Mod{Subtitle: "Refresh podcast", Valid: true, Icon: struct {
			Path string `json:"path"`
		}{Path: "icons/refresh.png"}}
		alt.SetVar("action", "refresh")
		alt.SetVar("podcast", e.Author)
		item.Mods.Alt = alt
		workflow.AddItem(item)
	}
	item := Item{
		Title:    "Go Back",
		Valid:    true,
		Icon:     struct{ Path string `json:"path"` }{Path: "icons/back.png"},
	}
	item.SetVar("trigger", "podcasts")
	workflow.AddItem(&item)
}

func ListQueue() {
	playlist, err := GetPlaylist()
	if err != nil || len(playlist) == 0 {
		workflow.WarnEmpty("No Episodes Found")
		return
	}
	var episodes []*Episode
	latestEpisodes := GetLatestEpisodes(false)
	for _, i := range playlist {
		if !i.Current && len(episodes) == 0 {
			continue
		}
		for _, e := range latestEpisodes {
			if e.URL == i.Filename {
				item := e.Format()
				item.Valid = !i.Current
				item.SetVar("action", "play")
				if len(episodes) > 1 {
					alt := Mod{Subtitle: "Play next", Valid:true, Icon: struct {Path string `json:"path"`}{Path: "icons/moveUp.png"}}
					alt.SetVar("action", "playNext")
					item.Mods.Alt = alt
				} else if len(episodes) == 0 {
					item.Mods.Cmd = Mod{}
				}
				ctrl := Mod{Subtitle: "Remove from queue", Valid:true, Icon: struct {Path string `json:"path"`}{Path: "icons/trash.png"}}
				ctrl.SetVar("action", "remove")
				ctrl.SetVar("url", e.URL)
				item.Mods.Ctrl = ctrl
				workflow.AddItem(item)
				episodes = append(episodes, e)
				break
			}
		}
	}
	if len(episodes) == 0 {
		workflow.WarnEmpty("No Episodes Found")
	} else {
		workflow.AddItem(PlayerControl(episodes))
	}
}

func (p *Podcast) Format() *Item {
	var icon = getCachePath("artworks", p.Name)
	_, err := os.Stat(icon); if err != nil {
		icon = ""
	}
	var item = Item{
		Title: p.Name,
		Subtitle: p.Desc,
		Match: matchString(p.Name),
		Valid: true,
		QuickLookURL: p.Link,
		Text: struct {
			Copy      string `json:"copy,omitempty"`
			LargeType string `json:"largetype,omitempty"`
		}{LargeType: p.Desc},
		Icon: struct {
			Path string `json:"path"`
		}{Path: icon},
		Mods: struct {
			Cmd   Mod `json:"cmd"`
			Alt   Mod `json:"alt"`
			Shift Mod `json:"shift"`
			Ctrl  Mod `json:"ctrl"`
			AltShift    Mod `json:"alt+shift"`
		}{},
	}
	item.SetVar("trigger", "episodes")
	item.SetVar("podcast", p.Name)

	alt := Mod{Subtitle: "Refresh podcast", Valid: true, Icon: struct {Path string `json:"path"`}{Path: "icons/refresh.png"}}
	alt.SetVar("action", "refresh")
	alt.SetVar("podcast", p.Name)
	item.Mods.Alt = alt

	altShift := Mod{Subtitle: "Refresh all podcasts", Valid: true, Icon: struct {
		Path string `json:"path"`
	}{Path: "icons/refresh.png"}}
	altShift.SetVar("action", "refreshAll")
	item.Mods.AltShift = altShift
	return &item
}

func (e *Episode) Format() *Item {
	var icon = getCachePath("artworks", e.Author)
	if _, err := os.Stat(icon); err != nil {
		icon = ""
	}
	var item = Item{
		Title:         e.Title,
		Subtitle:      fmt.Sprintf("🗓  %s  ·  ⌛️ %s", e.Date.Format("Mon, 2006-01-02"), formatDuration(e.Duration)),
		Valid:         true,
		QuickLookURL:  e.CacheShownote(),
		Icon: struct {
			Path string `json:"path"`
		}{Path: icon},
		Match:        matchString(e.Title, e.Author),
		AutoComplete: e.Author,
		Mods: struct {
			Cmd       Mod `json:"cmd"`
			Alt       Mod `json:"alt"`
			Shift     Mod `json:"shift"`
			Ctrl      Mod `json:"ctrl"`
			AltShift  Mod `json:"alt+shift"`
		}{},
	}
	item.SetVar("action", "addToQueue")
	item.SetVar("podcast", e.Author)
	item.SetVar("url", e.URL)

	cmd := Mod{Subtitle: "Play now", Valid: true, Icon: struct {Path string `json:"path"`}{Path: "icons/play.png"}}
	cmd.SetVar("action", "play")
	cmd.SetVar("url", e.URL)
	item.Mods.Cmd = cmd

	shift := Mod{Subtitle: "List " + e.Author, Valid: true,}
	shift.SetVar("trigger", "episodes")
	shift.SetVar("podcast", e.Author)
	item.Mods.Shift = shift
	return &item
}
