package main

import (
	"fmt"
	"log"
	"os"
)

var (
	cacheDir   = os.Getenv("alfred_workflow_cache")
	podcastMap map[string]*Podcast
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
	case "play_now":
		if err := PlayEpisode(os.Getenv("url"), ""); err != nil {
			Notify(err.Error(), "Error")
		}
	case "syncPlaylist":
		if playlist, err := generatePlaylist(); err == nil {
			loadPlaylist(playlist, "insert-next-play")
		}
	case "play_next", "play_last":
    p := &Podcast {
      UUID: os.Getenv("podcastUuid"),
    }
    p.GetEpisodes(false)
    if e, ok := p.EpisodeMap[os.Getenv("uuid")]; ok {
      if _, err := e.AddToQueue(action); err != nil {
        Notify(err.Error(), "Error")
      } else {
        Notify("Added to queue: " + e.Title)
      }
    } else {
      Notify("Episode not found", "Error")
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
		p := &Podcast{URL: os.Args[1]}
		if err := p.Subscribe(); err != nil {
			Notify(err.Error(), "Error")
		} else {
			Notify("Subscribed to " + p.Name)
		}
	case "unsubscribe":
		p := &Podcast{UUID: os.Getenv("podcastUuid")}
		p.GetEpisodes(false)
		if err := p.Unsubscribe(); err != nil {
			Notify(err.Error(), "Error")
		} else {
			Notify("Unsubscribed from " + p.Name)
			p.ClearCache()
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
		p := &Podcast{Name: os.Getenv("podcast"), UUID: os.Getenv("podcastUuid")}
		p.GetEpisodes(false)
		p.ListEpisodes()
	case "queue":
		ListUpNext()
	case "playing":
		GetPlaying()
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
		refreshCache([]string{os.Getenv("refresh"), os.Getenv("podcast")})
		workflow.SetVar("refresh", "")
	} else if action != "" {
		performAction(action)
		fmt.Println(`{"alfredworkflow":{"variables":{"action":""}}}`)
		return
	}

	workflow.SetVar("trigger", trigger)

	runTrigger(trigger)

	workflow.Output()
}
