package main_test

import (
	"testing"

	"github.com/twio142/alfred-podcasts"
)

func TestPlayEpisode(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		u        string
		position string
		wantErr  bool
	}{
		{
			name:    "play episode",
			u:       "https://dts-api.xiaoyuzhoufm.com/track/673699f98f10138dbc7808c7/67cdda65e924d4525a6b387b/media.xyzcdn.net/673699f98f10138dbc7808c7/lulrlShHTt8GRX3__W3UCGkGI5Hm.m4a",
			position: "",
			wantErr: false,
		},
		{
			name:    "play episode next",
			u:       "https://cdn.lizhi.fm/audio/2025/03/15/3132776168121244678_hd.mp3",
			position: "next",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := main.PlayEpisode(tt.u, tt.position)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("PlayEpisode() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("PlayEpisode() succeeded unexpectedly")
			}
		})
	}
}
