package main_test

import (
	"testing"

	"github.com/twio142/alfred-podcasts"
)

func TestListPodcasts(t *testing.T) {
	tests := []struct {
		name string // description of this test case
	}{
		{
			name: "ListPodcasts",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main.ListPodcasts()
		})
	}
}

func TestListNewReleases(t *testing.T) {
	tests := []struct {
		name string // description of this test case
	}{
		{
			name: "ListNewReleases",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main.ListNewReleases()
		})
	}
}

func TestPodcast_ListEpisodes(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		podcast main.Podcast
	}{
		{
			name: "ListEpisodes",
			podcast: main.Podcast{
				Name: "The Daily",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.podcast.GetEpisodes(false); err != nil {
				t.Errorf("Podcast.ListEpisodes() error = %v", err)
				return
			}
			tt.podcast.ListEpisodes("podcasts")
		})
	}
}

func TestListUpNext(t *testing.T) {
	tests := []struct {
		name string // description of this test case
	}{
		{
			name: "ListUpNext",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main.ListUpNext()
		})
	}
}

