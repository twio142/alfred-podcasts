package main

import (
	"os"
	"log"
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

	switch os.Getenv("action") {
	case "refresh":
		podcast = &Podcast{Name: os.Getenv("podcast")}
		podcast.GetEpisodes(true)
	case "refreshAll":
		GetAllPodcasts(true)
		os.Remove(getCachePath("latest"))
	case "refreshLatest":
		os.Remove(getCachePath("latest"))
	case "refreshInBackground":
		GetAllPodcasts(true)
		clearOldCache()
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
	case "15Back":
		runCommand("seek", "-15")
	case "next":
		runCommand("playlist-next")
	case "save":
		SavePlaylist()
		Notify("Playlist saved")
	default:
		// do nothing
	}

	workflow.SetVar("action", "")
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
		if title := os.Getenv("title"); title != "" {
			for _, e := range GetLatestEpisodes(false) {
				if e.Title == title {
					item := e.Format()
					item.Valid = false
					item.Mods.Cmd = Mod{}
					workflow.AddItem(item)
					break
				}
			}
		}
	case "test":
		log.Println("test")
	default:
	}

	workflow.Output()
}
