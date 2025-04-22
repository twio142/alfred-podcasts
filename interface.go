package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
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
		item := p.Format(false)
		workflow.AddItem(item)
	}
	workflow.SetVar("prevTrigger", "podcasts")
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
	GetUpNext(false)
	for _, e := range episodes {
		item := e.Format(false)
		// ⇧⌘ refresh new releases
		cmdShift := &Mod{Subtitle: "Refresh new releases", Icon: &Icon{Path: "icons/refresh.png"}}
		cmdShift.SetVar("refresh", "new_release")
		item.Mods.CmdShift = cmdShift
		item.Mods.Shift.SetVar("prevTrigger", "latest")
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
	for i, e := range episodes {
		item := e.Format(true)
		if i == 0 {
			item.Mods.Alt = nil
		}
		if i < 2 {
			item.Mods.Cmd = nil
		}
		item.Mods.Shift.SetVar("prevTrigger", "queue")
		workflow.AddItem(item)
	}
	upNextSummary(episodes)
}

func (p *Podcast) ListEpisodes(goBackTo string) {
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
	GetUpNext(false)
	for i, e := range episodes {
		if i == 30 {
			break
		}
		item := e.Format(false)
		item.Subtitle = fmt.Sprintf("􀪔 %s  ·  %s", e.Podcast, item.Subtitle)
		item.Match = matchString(e.Title)
		item.AutoComplete = ""
		// ⇧⌘ refresh podcast
		cmdShift := &Mod{Subtitle: "Refresh podcast", Icon: &Icon{Path: "icons/refresh.png"}}
		cmdShift.SetVar("refresh", "podcast")
		cmdShift.SetVar("podcastUuid", e.PodcastUUID)
		item.Mods.CmdShift = cmdShift
		item.Mods.Shift = nil
		item.Mods.Ctrl = nil
		item.Mods.Fn = nil
		workflow.AddItem(item)
	}
	item := Item{
		Title: "Go Back",
		Icon:  &Icon{Path: "icons/back.png"},
	}
	item.SetVar("trigger", goBackTo)
	workflow.AddItem(&item)
	workflow.SetVar("prevTrigger", "")
}

func (p *Podcast) Format(search bool) *Item {
	icon := &Icon{Path: getCachePath("artworks", p.UUID)}
	_, err := os.Stat(icon.Path)
	if err != nil {
		icon = nil
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
		Icon: icon,
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

	if search {
		item.Subtitle = "􀊱 " + p.Author
		// ⌘ subscribe / unsubsribe podcast
		var cmd *Mod
		if _, ok := podcastMap[p.UUID]; ok {
			item.Title = "􀁢 " + item.Title
			cmd = &Mod{Subtitle: "Unsubscribe", Icon: &Icon{Path: "icons/trash.png"}}
			cmd.SetVar("actionKeep", "unsubscribe")
		} else {
			cmd = &Mod{Subtitle: "Subscribe", Icon: &Icon{Path: "icons/plus.png"}}
			cmd.SetVar("actionKeep", "subscribe")
		}
		cmd.SetVar("podcastUuid", p.UUID)
		cmd.SetVar("podcast", p.Name)
		item.Mods.Cmd = cmd
	} else {
		// ⌘ refresh podcast
		cmd := &Mod{Subtitle: "Refresh podcast", Icon: &Icon{Path: "icons/refresh.png"}}
		cmd.SetVar("refresh", "podcast")
		cmd.SetVar("podcastUuid", p.UUID)
		item.Mods.Cmd = cmd

		// ⇧⌘ refresh all podcasts
		cmdShift := &Mod{Subtitle: "Refresh all podcasts", Icon: &Icon{Path: "icons/refresh.png"}}
		cmdShift.SetVar("refresh", "allPodcasts")
		item.Mods.CmdShift = cmdShift

		// ⌃ unsubscribe podcast
		ctrl := &Mod{Subtitle: "Unsubscribe", Icon: &Icon{Path: "icons/trash.png"}}
		ctrl.SetVar("action", "unsubscribe")
		ctrl.SetVar("podcastUuid", p.UUID)
		item.Mods.Ctrl = ctrl
	}
	return &item
}

func (e *Episode) Format(upNext bool) *Item {
	icon := &Icon{Path: getCachePath("artworks", e.PodcastUUID)}
	if _, err := os.Stat(icon.Path); err != nil {
		icon = nil
	}
	if e.Duration == 0 || e.ShowNotes == "" {
		p := &Podcast{UUID: e.PodcastUUID}
		if err := p.GetEpisodes(false); err == nil {
			if _e, ok := p.EpisodeMap[e.UUID]; ok {
				e.Duration = _e.Duration
				e.ShowNotes = _e.ShowNotes
				e.Date = _e.Date
				e.Image = _e.Image
			}
		}
	}
	item := Item{
		Title:        e.Title,
		Subtitle:     fmt.Sprintf("􀉉 %s  ·  􀖈 %s", e.Date.Format("Mon, 2006-01-02"), formatDuration(e.Duration)),
		Arg:          e.URL,
		UID:          e.UUID,
		QuickLookURL: e.CacheShownotes(),
		Icon:         icon,
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
	action := "action"
	if !upNext {
		if _, ok := upNextMap[e.UUID]; ok {
			item.Title = "􀑬 " + item.Title
		}
		action = "actionKeep"
		// ↵ add episode to end of queue
		item.SetVar(action, "play_last")
		item.SetVar("uuid", e.UUID)
		item.SetVar("podcastUuid", e.PodcastUUID)
	}

	// ⌘ add episode to top of queue
	cmd := &Mod{Subtitle: "Play next", Icon: &Icon{Path: "icons/playNext.png"}}
	cmd.SetVar(action, "play_next")
	cmd.SetVar("uuid", e.UUID)
	cmd.SetVar("podcastUuid", e.PodcastUUID)
	item.Mods.Cmd = cmd

	// ⌥ play episode now
	alt := &Mod{Subtitle: "Play now", Icon: &Icon{Path: "icons/play.png"}}
	alt.SetVar(action, "play_now")
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
	ctrl.SetVar(action, "markAsPlayed")
	ctrl.SetVar("uuid", e.UUID)
	ctrl.SetVar("podcastUuid", e.PodcastUUID)
	item.Mods.Ctrl = ctrl

	//  archive episode
	fn := &Mod{Subtitle: "Archive", Icon: &Icon{Path: "icons/archive.png"}}
	fn.SetVar(action, "archive")
	fn.SetVar("uuid", e.UUID)
	fn.SetVar("podcastUuid", e.PodcastUUID)
	item.Mods.Fn = fn

	return &item
}

func upNextSummary(episodes []*Episode) {
	if len(episodes) == 0 {
		return
	}
	var totalDuration int
	for _, e := range episodes {
		totalDuration += e.Duration - e.PlayedUpTo
	}
	item := Item{
		Title:    fmt.Sprintf("%d Episodes, %s Remaining", len(episodes), formatDuration(totalDuration)),
	}
	// ⌘ refresh queue
	cmd := &Mod{Subtitle: "Refresh queue", Icon: &Icon{Path: "icons/refresh.png"}}
	cmd.SetVar("refresh", "up_next")
	item.Mods.Cmd = cmd

	// ⌥ replace playlist
	alt := &Mod{Subtitle: "Replace playlist", Icon: &Icon{Path: "icons/play.png"}}
	alt.SetVar("actionKeep", "replace")
	item.Mods.Alt = alt

	// ⇧⌥ append playlist
	altShift := &Mod{Subtitle: "Append playlist", Icon: &Icon{Path: "icons/play.png"}}
	altShift.SetVar("actionKeep", "insert-next-play")
	item.Mods.AltShift = altShift

	// ⇧ sync playlist
	shift := &Mod{Subtitle: "Sync playlist", Icon: &Icon{Path: "icons/sync.png"}}
	shift.SetVar("action", "sync")
	item.Mods.Shift = shift
	workflow.UnshiftItem(&item)
}

func Search(query string) error {
	var searchResults []*Podcast
	var searchErr, listErr error
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		listErr = GetPodcastList(false)
	}()
	go func() {
		defer wg.Done()
		if query == "" {
			file := getCachePath("search_results")
			if data, err := readCache(file, time.Duration(math.MaxInt64)); err != nil {
				searchErr = err
			} else if err := json.Unmarshal(data, &searchResults); err != nil {
				searchErr = err
			}
		} else {
			searchResults, searchErr = SearchPodcasts(query)
		}
	}()
	wg.Wait()

	if listErr != nil {
		workflow.WarnEmpty(listErr.Error())
		return listErr
	}
	if searchErr != nil {
		workflow.WarnEmpty(searchErr.Error())
		return searchErr
	}
	if len(searchResults) == 0 {
		workflow.WarnEmpty("No Podcasts Found")
		return nil
	}
	for _, p := range searchResults {
		item := p.Format(true)
		item.Mods.Shift.SetVar("prevTrigger", "search")
		workflow.AddItem(item)
	}
	return nil
}
