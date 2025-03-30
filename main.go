package main

import (
	"fmt"
	"log"
	"os"
)

var (
	cacheDir   = os.Getenv("alfred_workflow_cache")
	podcastMap map[string]*Podcast
	upNextMap  map[string]*Episode
	workflow   = Workflow{}
)

func setup() {
	if _, err := os.Stat(cacheDir + "/podcasts"); os.IsNotExist(err) {
		if err = os.MkdirAll(cacheDir+"/podcasts", 0755); err != nil {
			log.Fatal(err)
		}
	}
	if _, err := os.Stat(cacheDir + "/artworks"); os.IsNotExist(err) {
		if err = os.MkdirAll(cacheDir+"/artworks", 0755); err != nil {
			log.Fatal(err)
		}
	}
	if _, err := os.Stat(cacheDir + "/shownotes"); os.IsNotExist(err) {
		if err = os.MkdirAll(cacheDir+"/shownotes", 0755); err != nil {
			log.Fatal(err)
		}
	}
}

func performAction(action string) {
	switch action {
	case "insert-next-play", "replace":
		if playlist, err := ExportPlaylist(); err == nil {
			loadPlaylist(playlist, action)
		}
	case "play_now", "play_next", "play_last":
		p := &Podcast{
			UUID: os.Getenv("podcastUuid"),
		}
		p.GetEpisodes(false)
		if e, ok := p.EpisodeMap[os.Getenv("uuid")]; ok {
			if _, err := e.AddToQueue(action); err != nil {
				Notify(err.Error(), "Error")
			} else if action == "play_now" {
				if playlist, err := ExportPlaylist(); err == nil {
					loadPlaylist(playlist, "replace")
				}
			} else {
				Notify("Added to queue: " + e.Title)
			}
		} else {
			Notify("Episode not found", "Error")
		}
	case "sync":
		if err := SyncPlaylist(); err != nil {
			Notify(err.Error(), "Error")
		}
	case "markAsPlayed", "archive":
		e := &Episode{UUID: os.Getenv("uuid"), PodcastUUID: os.Getenv("podcastUuid")}
		if err := e.Archive(action == "markAsPlayed"); err != nil {
			Notify(err.Error(), "Error")
		} else if action == "markAsPlayed" {
			Notify("Marked as played: " + e.Title)
		} else {
			Notify("Archived: " + e.Title)
		}
	case "subscribe":
		p := &Podcast{UUID: os.Getenv("podcastUuid"), Name: os.Getenv("podcast")}
		if err := p.Subscribe(); err != nil {
			Notify(err.Error(), "Error")
		} else {
			if p.Name == "" {
				p.GetInfo()
			}
			Notify("Subscribed to " + p.Name)
			GetPodcastList(true)
		}
	case "unsubscribe":
		p := &Podcast{UUID: os.Getenv("podcastUuid"), Name: os.Getenv("podcast")}
		if p.Name == "" {
			p.GetInfo()
		}
		if err := p.Unsubscribe(); err != nil {
			Notify(err.Error(), "Error")
		} else {
			Notify("Unsubscribed from " + p.Name)
			p.ClearCache()
			GetPodcastList(true)
		}
	default:
		// do nothing
	}
}

func runTrigger(trigger string) {
	switch trigger {
	case "podcasts":
		ListPodcasts()
	case "latest":
		ListNewReleases()
	case "episodes":
		p := &Podcast{UUID: os.Getenv("podcastUuid")}
		p.GetEpisodes(false)
		p.ListEpisodes()
	case "queue":
		ListUpNext()
	case "playing":
		GetPlaying()
	case "search":
		Search(os.Args[1])
	case "test":
		log.Println("test")
	default:
	}
}

func main() {
	setup()

	trigger := os.Getenv("trigger")
	action := os.Getenv("action")
	if action == "" {
		action = os.Getenv("actionKeep")
	}

	if os.Getenv("refresh") != "" {
		refreshCache([]string{os.Getenv("refresh"), os.Getenv("podcastUuid")})
		fmt.Println(`{"alfredworkflow":{"variables":{"refresh":""}}}`)
		return
	} else if action != "" {
		performAction(action)
		fmt.Println(`{"alfredworkflow":{"variables":{"action":""}}}`)
		return
	}

	workflow.SetVar("trigger", trigger)

	runTrigger(trigger)

	workflow.Output()
}
