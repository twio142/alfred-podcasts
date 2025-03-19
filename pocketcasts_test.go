package main_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/twio142/alfred-podcasts"
)

func TestMain(m *testing.M) {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}
	os.Exit(m.Run())
}

func TestPocketCastsLogin(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantErr  bool
	}{
		{
			name:     "valid login",
			email:    os.Getenv("email"),
			password: os.Getenv("password"),
			wantErr:  false,
		},
		{
			name:     "invalid login",
			email:    "invalid",
			password: "invalid",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := main.PocketCastsLogin(tt.email, tt.password)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("PocketCastsLogin() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("PocketCastsLogin() succeeded unexpectedly")
			}
		})
	}
}

func TestPocketCastsGetPodcasts(t *testing.T) {
	tests := []struct {
		name    string
		force   bool
		wantErr bool
	}{
		{
			name:    "valid get podcasts",
			force:   true,
			wantErr: false,
		},
		{
			name:    "valid read podcasts cache",
			force:   false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := main.PocketCastsGetPodcasts(tt.force)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("PocketCastsGetPodcasts() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("PocketCastsGetPodcasts() succeeded unexpectedly")
				return
			}
		})
	}
}

func TestPocketCastsGetUpNext(t *testing.T) {
	tests := []struct {
		name    string
		force   bool
		wantErr bool
	}{
		{
			name:    "valid get up next",
			force:   true,
			wantErr: false,
		},
		{
			name:    "valid read up next cache",
			force:   false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := main.PocketCastsGetUpNext(tt.force)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("PocketCastsGetUpNext() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("PocketCastsGetUpNext() succeeded unexpectedly")
			}
			if true {
				fmt.Printf("PocketCastsGetUpNext() = %v", got)
			}
		})
	}
}

func TestPocketCastsGetList(t *testing.T) {
	tests := []struct {
		name    string
		force   bool
		list    string
		wantErr bool
	}{
		{
			name:    "valid get new releases",
			force:   true,
			list:    "new_releases",
			wantErr: false,
		},
		{
			name:    "valid read new releases cache",
			force:   false,
			list:    "new_releases",
			wantErr: false,
		},
		{
			name:    "valid get history",
			force:   true,
			list:    "history",
			wantErr: false,
		},
		{
			name:    "valid read history cache",
			force:   false,
			list:    "history",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := main.PocketCastsGetList(tt.list, tt.force)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("PocketCastsGetNewReleases() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("PocketCastsGetNewReleases() succeeded unexpectedly")
			}
			if true {
				fmt.Printf("PocketCastsGetNewReleases() = %v", got)
			}
		})
	}
}

func TestPodcast_PocketCastsGetEpisodes(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		podcast main.Podcast
		// Named input parameters for target function.
		force   bool
		wantErr bool
	}{
    {
      name: "valid get podcast episodes",
      podcast: main.Podcast{
        Name: "The Daily",
      },
      force: true,
      wantErr: false,
    },
		{
			name: "valid read podcast episodes cache",
			podcast: main.Podcast{
				Name: "The Daily",
			},
			force:   false,
			wantErr: false,
		},
    {
      name: "invalid get podcast episodes",
      podcast: main.Podcast{},
      force: true,
      wantErr: true,
    },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := tt.podcast.PocketCastsGetEpisodes(tt.force)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("PocketCastsGetPodcastEpisodes() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("PocketCastsGetPodcastEpisodes() succeeded unexpectedly")
			}
		})
	}
}

