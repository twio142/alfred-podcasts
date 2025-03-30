package main_test

import (
	"fmt"
	"testing"

	"github.com/twio142/alfred-podcasts"
)

func TestGetAllPodcasts(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		force   bool
		wantErr bool
	}{
		{
			name:    "force get all podcasts",
			force:   true,
			wantErr: false,
		},
		{
			name:    "get all podcasts",
			force:   false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := main.GetAllPodcasts(tt.force)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("GetAllPodcasts() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("GetAllPodcasts() succeeded unexpectedly")
			}
		})
	}
}

func TestPodcast_CacheArtwork(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		podcast  *main.Podcast
	}{
		{
			name:    "cache artwork",
			podcast: &main.Podcast{
				UUID: "05a51e00-7d3d-013d-2494-0eea28d86ca3",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.podcast.CacheArtwork()
		})
	}
}

func TestEpisode_CacheShownotes(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		podcast  *main.Podcast
	}{
		{
			name: "cache shownotes",
			podcast: &main.Podcast{
				UUID: "05a51e00-7d3d-013d-2494-0eea28d86ca3",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.podcast.GetEpisodes(false); err != nil {
				t.Fatalf("GetEpisodes() failed: %v", err)
			}
			for _, e := range tt.podcast.EpisodeMap {
				e.CacheShownotes()
			}
		})
	}
}

func TestExportPlaylist(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		wantErr bool
	}{
		{
			name:    "export playlist",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := main.ExportPlaylist()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ExportPlaylist() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ExportPlaylist() succeeded unexpectedly")
			}
			if true {
				fmt.Printf("ExportPlaylist() = %v", got)
			}
		})
	}
}

func TestSyncPlaylist(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		wantErr bool
	}{
		{
			name:    "sync playlist",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := main.SyncPlaylist()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("SyncPlaylist() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("SyncPlaylist() succeeded unexpectedly")
			}
		})
	}
}

