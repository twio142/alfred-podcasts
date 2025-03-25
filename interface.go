package main

import (
	"fmt"
	"os"
	"sort"
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
			Subtitle: "Refresh",
			Icon:     &Icon{Path: "icons/refresh.png"},
		}
		item.SetVar("refresh", "new_releases")
		workflow.AddItem(&item)
		return
	}
	for i, e := range episodes {
		item := e.Format()
		if i == 1 {
			item.Mods.Alt = nil
		}
		if i < 3 {
			item.Mods.Cmd = nil
		}
		//  refresh new releases
		fn := &Mod{Subtitle: "Refresh new releases", Icon: &Icon{Path: "icons/refresh.png"}}
		fn.SetVar("refresh", "new_release")
		item.Mods.Fn = fn
		workflow.AddItem(item)
	}
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
			Subtitle: "Refresh",
			Icon:     &Icon{Path: "icons/refresh.png"},
		}
		item.SetVar("refresh", "up_next")
		workflow.AddItem(&item)
		return
	}
	for _, e := range episodes {
		item := e.Format()
		// ↵ do nothing
		item.SetVar("actionKeep", "noop")
		workflow.AddItem(item)
	}
	upNextSummary(episodes)
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
		//  refresh podcast
		fn := &Mod{Subtitle: "Refresh podcast", Icon: &Icon{Path: "icons/refresh.png"}}
		fn.SetVar("refresh", "podcast")
		fn.SetVar("podcastUuid", e.PodcastUUID)
		item.Mods.Fn = fn
		item.Mods.Shift = nil
		item.Mods.Ctrl = nil
		item.Mods.CtrlShift = nil
		workflow.AddItem(item)
	}
	item := Item{
		Title: "Go Back",
		Icon:  &Icon{Path: "icons/back.png"},
	}
	item.SetVar("trigger", "podcasts")
	workflow.AddItem(&item)
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
			Cmd       *Mod `json:"cmd,omitempty"`
			Alt       *Mod `json:"alt,omitempty"`
			Shift     *Mod `json:"shift,omitempty"`
			Ctrl      *Mod `json:"ctrl,omitempty"`
			Fn        *Mod `json:"fn,omitempty"`
			AltShift  *Mod `json:"alt+shift,omitempty"`
			CtrlShift *Mod `json:"ctrl+shift,omitempty"`
			CmdShift  *Mod `json:"cmd+shift,omitempty"`
		}{},
	}

	// ↵ list episodes
	item.SetVar("trigger", "episodes")
	item.SetVar("podcastUuid", p.UUID)

	// ⌥ refresh podcast
	alt := &Mod{Subtitle: "Refresh podcast", Icon: &Icon{Path: "icons/refresh.png"}}
	alt.SetVar("refresh", "podcast")
	alt.SetVar("podcastUuid", p.UUID)
	item.Mods.Alt = alt

	// ⇧⌥ refresh all podcasts
	altShift := &Mod{Subtitle: "Refresh all podcasts", Icon: &Icon{Path: "icons/refresh.png"}}
	altShift.SetVar("refresh", "allPodcasts")
	item.Mods.AltShift = altShift

	// ⌃ unsubscribe podcast
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
	if e.Duration == 0 {
		p := &Podcast{UUID: e.PodcastUUID}
		if err := p.GetEpisodes(false); err == nil {
			if _e, ok := p.EpisodeMap[e.UUID]; ok {
				e.Duration = _e.Duration
				e.ShowNotes = _e.ShowNotes
				e.Date = _e.Date
			}
		}
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
			Cmd       *Mod `json:"cmd,omitempty"`
			Alt       *Mod `json:"alt,omitempty"`
			Shift     *Mod `json:"shift,omitempty"`
			Ctrl      *Mod `json:"ctrl,omitempty"`
			Fn        *Mod `json:"fn,omitempty"`
			AltShift  *Mod `json:"alt+shift,omitempty"`
			CtrlShift *Mod `json:"ctrl+shift,omitempty"`
			CmdShift  *Mod `json:"cmd+shift,omitempty"`
		}{},
	}
	// ↵ add episode to end of queue
	item.SetVar("actionKeep", "play_last")
	item.SetVar("uuid", e.UUID)
	item.SetVar("podcastUuid", e.PodcastUUID)

	// ⌘ add episode to top of queue
	cmd := &Mod{Subtitle: "Play next", Icon: &Icon{Path: "icons/playNext.png"}}
	cmd.SetVar("actionKeep", "play_next")
	cmd.SetVar("uuid", e.UUID)
	cmd.SetVar("podcastUuid", e.PodcastUUID)
	item.Mods.Cmd = cmd

	// ⌥ play episode now
	alt := &Mod{Subtitle: "Play now", Icon: &Icon{Path: "icons/play.png"}}
	alt.SetVar("actionKeep", "play_now")
	alt.SetVar("uuid", e.UUID)
	alt.SetVar("podcastUuid", e.PodcastUUID)
	item.Mods.Alt = alt

	// ⇧ list episodes of this podcast
	shift := &Mod{Subtitle: "􀪔 " + e.Podcast}
	shift.SetVar("trigger", "episodes")
	shift.SetVar("podcastUuid", e.PodcastUUID)
	item.Mods.Shift = shift

	// ⌃ mark episode as played
	ctrl := &Mod{Subtitle: "Mark as played", Icon: &Icon{Path: "icons/check.png"}}
	ctrl.SetVar("actionKeep", "markAsPlayed")
	ctrl.SetVar("uuid", e.UUID)
	ctrl.SetVar("podcastUuid", e.PodcastUUID)
	item.Mods.Ctrl = ctrl

	// ⌃⇧ archive episode
	ctrlShift := &Mod{Subtitle: "Archive", Icon: &Icon{Path: "icons/archive.png"}}
	ctrlShift.SetVar("actionKeep", "archive")
	ctrlShift.SetVar("uuid", e.UUID)
	ctrlShift.SetVar("podcastUuid", e.PodcastUUID)
	item.Mods.CtrlShift = ctrlShift

	return &item
}

func upNextSummary(episodes []*Episode) {
	if len(episodes) == 0 {
		return
	}
	var totalDuration int
	for _, e := range episodes {
		totalDuration += e.Duration
	}
	item := Item{
		Title:    fmt.Sprintf("%d Episodes, %s Remaining", len(episodes), formatDuration(totalDuration)),
		Subtitle: "Append playlist",
		Icon:     &Icon{Path: "icons/play.png"},
	}
	// ↵ append playlist
	item.SetVar("actionKeep", "insert-next-play")

	// ⌘ replace playlist
	cmd := &Mod{Subtitle: "Replace playlist", Icon: &Icon{Path: "icons/play.png"}}
	cmd.SetVar("action", "replace")
	item.Mods.Cmd = cmd

	// ⌥ refresh queue
	alt := &Mod{Subtitle: "Refresh queue", Icon: &Icon{Path: "icons/refresh.png"}}
	alt.SetVar("refresh", "up_next")
	item.Mods.Alt = alt
	workflow.UnshiftItem(&item)
}
