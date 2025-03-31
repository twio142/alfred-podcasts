package main_test

import (
	"fmt"
	"testing"

	"github.com/twio142/alfred-podcasts"
)

func TestEpisode_QueueActions(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		episode *main.Episode
		// Named input parameters for target function.
		action  string
		wantErr bool
	}{
		{
			name: "valid play next",
			episode: &main.Episode{
				UUID:        "87ecf9da-1122-4ffd-98c0-c150a42c3268",
				PodcastUUID: "4eb5b260-c933-0134-10da-25324e2a541d",
				Title:       "'The Interview': Rick Steves Refuses To Get Cynical About the World",
				URL:         "https://dts.podtrac.com/redirect.mp3/pdst.fm/e/pfx.vpixl.com/6qj4J/nyt.simplecastaudio.com/03d8b493-87fc-4bd1-931f-8a8e9b945d8a/episodes/56da6caf-0cf0-43e1-a867-9f00a6c29ba9/audio/128/default.mp3?aid=rss_feed&awCollectionId=03d8b493-87fc-4bd1-931f-8a8e9b945d8a&awEpisodeId=56da6caf-0cf0-43e1-a867-9f00a6c29ba9&feed=54nAGcIl",
			},
			action:  "play_next",
			wantErr: false,
		},
		{
			name: "valid play last",
			episode: &main.Episode{
				UUID:        "87ecf9da-1122-4ffd-98c0-c150a42c3268",
				PodcastUUID: "4eb5b260-c933-0134-10da-25324e2a541d",
				Title:       "A Turning Point for Ultraprocessed Foods",
				URL:         "https://dts.podtrac.com/redirect.mp3/pdst.fm/e/pfx.vpixl.com/6qj4J/nyt.simplecastaudio.com/03d8b493-87fc-4bd1-931f-8a8e9b945d8a/episodes/75dfb7a8-2d29-4ff8-91de-9fc56ece08b9/audio/128/default.mp3?aid=rss_feed&awCollectionId=03d8b493-87fc-4bd1-931f-8a8e9b945d8a&awEpisodeId=75dfb7a8-2d29-4ff8-91de-9fc56ece08b9&feed=54nAGcIl",
			},
			action:  "play_last",
			wantErr: false,
		},
		{
			name: "valid play now",
			episode: &main.Episode{
				UUID:        "87ecf9da-1122-4ffd-98c0-c150a42c3268",
				PodcastUUID: "4eb5b260-c933-0134-10da-25324e2a541d",
				Title:       "'The Interview': Rick Steves Refuses To Get Cynical About the World",
				URL:         "https://dts.podtrac.com/redirect.mp3/pdst.fm/e/pfx.vpixl.com/6qj4J/nyt.simplecastaudio.com/03d8b493-87fc-4bd1-931f-8a8e9b945d8a/episodes/56da6caf-0cf0-43e1-a867-9f00a6c29ba9/audio/128/default.mp3?aid=rss_feed&awCollectionId=03d8b493-87fc-4bd1-931f-8a8e9b945d8a&awEpisodeId=56da6caf-0cf0-43e1-a867-9f00a6c29ba9&feed=54nAGcIl",
			},
			action:  "play_now",
			wantErr: false,
		},
		{
			name: "valid remove from queue",
			episode: &main.Episode{
				UUID: "edb628c3-1779-43ef-b6a7-a5d19d312e5b",
			},
			action:  "remove",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []*main.Episode
			var gotErr error
			switch tt.action {
			case "remove":
				got, gotErr = main.RemoveEpisodesFromQueue([]*main.Episode{tt.episode})
			default:
				got, gotErr = tt.episode.AddToQueue(tt.action)
			}
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("%s failed: %v", tt.name, gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatalf("%s succeeded unexpectedly", tt.name)
			}
			if true {
				fmt.Printf("%s: %v", tt.name, got)
			}
		})
	}
}

func TestEpisode_Archive(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		episode *main.Episode
		// Named input parameters for target function.
		markAsPlayed bool
		wantErr      bool
	}{
		{
			name: "valid archive",
			episode: &main.Episode{
				UUID:        "2753add2-b0cb-4e42-b5e8-4656e89cb478",
				PodcastUUID: "fe3d4040-10fa-0138-9f84-0acc26574db2",
			},
			markAsPlayed: true,
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := tt.episode.Archive(tt.markAsPlayed)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Archive() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Archive() succeeded unexpectedly")
			}
		})
	}
}

func TestSearchPodcasts(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		term    string
		wantErr bool
	}{
		{
			name:    "valid search",
			term:    "The Daily",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := main.SearchPodcasts(tt.term)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("SearchPodcasts() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("SearchPodcasts() succeeded unexpectedly")
			}
			if true {
				for _, podcast := range got {
					fmt.Println(podcast.Name)
					fmt.Println(podcast.UUID)
				}
			}
		})
	}
}

func TestPodcast_Subscribe(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		podcast *main.Podcast
		wantErr bool
	}{
		{
			name: "valid subscribe",
			podcast: &main.Podcast{
				UUID: "93b26340-a2cb-013a-d895-0acc26574db2",
			},
			wantErr: false,
		},
		{
			name: "valid subscribe",
			podcast: &main.Podcast{
				URL: "https://justpodmedia.com/rss/tipsy-proof.xml",
			},
			wantErr: false,
		},
		{
			name: "invalid subscribe",
			podcast: &main.Podcast{
				URL: "https://justpodmedia.com/rss/nonsense.xml",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := tt.podcast.Subscribe()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Subscribe() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Subscribe() succeeded unexpectedly")
			}
			if true {
				fmt.Printf("Subscribe() = %v", tt.podcast)
			}
		})
	}
}

func TestPodcast_Unsubscribe(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		podcast *main.Podcast
		wantErr bool
	}{
		{
			name: "valid unsubscribe",
			podcast: &main.Podcast{
				UUID: "8dd88e30-d447-0137-1e22-0acc26574db2",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := tt.podcast.Unsubscribe()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Unsubscribe() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Unsubscribe() succeeded unexpectedly")
			}
		})
	}
}
