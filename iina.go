package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

func runCommand(command ...any) (any, error) {
	if len(command) == 0 {
		return "", fmt.Errorf("no command provided")
	}
	cmd := exec.Command("socat", "-", "/tmp/iina.sock")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	go func() {
		defer stdin.Close()
		json.NewEncoder(stdin).Encode(map[string]any{
			"command": command,
		})
	}()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	var result struct {
		Data  any    `json:"data"`
		Error string `json:"error"`
	}
	if err = json.Unmarshal(out, &result); err != nil {
		return "", err
	}
	if result.Error != "success" {
		return "", fmt.Errorf("%v", result.Error)
	}
	return result.Data, nil
}

func PlayEpisode(u string, position string) error {
	if u == "" {
		return fmt.Errorf("no episode URL provided")
	}
	currentPos, err := runCommand("get_property", "playlist-current-pos")
	if err != nil {
		cmd := exec.Command("/usr/bin/open", "iina://weblink?url="+url.QueryEscape(u))
		return cmd.Run()
	}
	// NOTE: the flags `insert-*` only work since mpv 0.38.0
	switch position {
	case "next":
		_, err := runCommand("loadfile", u, "insert-next")
		return err
	case "last":
		_, err := runCommand("loadfile", u, "append")
		return err
	default:
		if _, err := runCommand("loadfile", u, "insert-at", currentPos); err != nil {
			return err
		}
		_, err := runCommand("playlist-play-index", currentPos)
		return err
	}
}

func PlayPause(p ...bool) error {
	if len(p) > 0 {
		pause := "no"
		if !p[0] {
			pause = "yes"
		}
		_, err := runCommand("set", "pause", pause)
		return err
	} else {
		_, err := runCommand("cycle", "pause")
		return err
	}
}

func loadPlaylist(file string, flag ...string) error {
	args := []any{"loadlist", file}
	for _, f := range flag {
		args = append(args, f)
	}
	if _, err := runCommand(args...); err != nil {
		cmd := exec.Command("/usr/bin/open", "-a", "IINA", file)
		return cmd.Run()
	}
	return nil
}

type PlaylistItem struct {
	Filename     string `json:"filename"`
	Current      bool   `json:"current"`
	PlaylistPath string `json:"playlist-path"`
}

func readPlaylist() (map[string]*Episode, error) {
	playlistPath := getCachePath("podcast_playlist.m3u")
	content, err := os.ReadFile(playlistPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read playlist: %w", err)
	}
	lines := strings.Split(string(content), "\n")
	episodeMap := make(map[string]*Episode)
	var currentEpisode *Episode

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			parts := strings.SplitN(line[2:], "\t", 2)
			if len(parts) != 2 {
				continue
			}
			currentEpisode = FindEpisode(map[string]string{"title": parts[1], "podcast": parts[0]})
		} else if currentEpisode != nil {
			episodeMap[line] = currentEpisode
			currentEpisode = nil
		}
	}
	return episodeMap, nil
}

func getPlaybackState() ([]*Episode, error) {
	playlist, err := runCommand("get_property", "playlist")
	if err != nil {
		return nil, err
	}
	var items []PlaylistItem
	playlistData, ok := playlist.([]any)
	if !ok {
		return nil, fmt.Errorf("no playlist found")
	}
	data, err := json.Marshal(playlistData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal playlist: %w", err)
	}
	if err = json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to unmarshal playlist: %w", err)
	}

	episodeMap, err := readPlaylist()
	if err != nil {
		return nil, err
	}
	var episodes []*Episode
	for _, item := range items {
		e, ok := episodeMap[item.Filename]
		if !ok {
			continue
		}
		episodes = append(episodes, e)
		if item.Current {
			if timePos, err := runCommand("get_property", "time-pos"); err == nil {
				if pos, ok := timePos.(float64); ok {
					e.PlayedUpTo = int(pos)
				}
			}
			break
		} else {
			e.Played = true
		}
	}
	return episodes, nil
}
