package main

import (
	"log"
	"fmt"
	"os"
)

var cacheDir = os.Getenv("alfred_workflow_cache")
var allPodcasts []*Podcast
var podcast *Podcast
var workflow = Workflow{}

func main() {
	if _, err := os.Stat(cacheDir + "/podcasts"); os.IsNotExist(err) {
		os.MkdirAll(cacheDir+"/podcasts", 0755)
	}
	if _, err := os.Stat(cacheDir + "/artworks"); os.IsNotExist(err) {
		os.MkdirAll(cacheDir+"/artworks", 0755)
	}
	if _, err := os.Stat(cacheDir + "/shownotes"); os.IsNotExist(err) {
		os.MkdirAll(cacheDir+"/shownotes", 0755)
	}

	trigger := os.Getenv("trigger")
	url := os.Getenv("url")
	action := os.Getenv("action")
	if action == "" {
		action = os.Getenv("actionKeep")
	}

	switch action {
	case "refresh":
		podcast = &Podcast{Name: os.Getenv("podcast")}
		podcast.GetEpisodes(true)
	case "refreshAll":
		GetAllPodcasts(true)
		refreshLatest()
	case "refreshLatest":
		refreshLatest()
	case "refreshInBackground":
		GetAllPodcasts(true)
		clearOldCache()
		defer os.Remove(getCachePath("podcasts.lock"))
	case "addToQueue":
		if err := AddToPlaylist(url); err != nil {
			Notify("Error", err.Error())
		} else if trigger != "queue" {
			AddToLatest(url, os.Getenv("podcast"))
		}
	case "play":
		if err := PlayEpisode(url); err != nil {
			Notify("Error", err.Error())
		}
	case "playNext":
		if err := PlayEpisode(url, true); err != nil {
			Notify("Error", err.Error())
		}
	case "remove":
		if err := RemoveFromPlaylist(url); err != nil {
			Notify("Error", err.Error())
		}
	case "playPause":
		PlayPause()
	case "30Back":
		runCommand("seek", "-30")
	case "next":
		runCommand("playlist-next")
	case "save":
		SavePlaylist()
		Notify("Playlist saved")
	case "loadList":
		if err := LoadPlaylist(); err != nil {
			Notify("Error", err.Error())
		}
	default:
		// do nothing
	}

	if action != "" {
		fmt.Println("{\"alfredworkflow\":{\"variables\":{\"action\":\"\"}}}")
		return
	}

	workflow.SetVar("trigger", trigger)

	switch trigger {
	case "podcasts":
		ListPodcasts()
	case "latest":
		ListLatest()
	case "episodes":
		ListEpisodes()
	case "queue":
		ListQueue()
	case "playing":
		if e := FindEpisode(map[string]string{"title": os.Getenv("title"), "author": os.Getenv("artist")}); e != nil {
			item := e.Format()
			valid := false
			item.Valid = &valid
			item.Mods.Cmd = nil
			workflow.AddItem(item)
		} else {
			workflow.WarnEmpty("No Episode Found")
		}
	case "test":
		log.Println("test")
	default:
	}

	workflow.Output()
}
