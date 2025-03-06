package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func runCommand(command ...string) (any, error) {
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

type PlaylistItem struct {
	Filename string `json:"filename"`
	Current  bool   `json:"current"`
	Playing  bool   `json:"playing"`
	Title    string `json:"title"`
	ID       int    `json:"id"`
}

func GetPlaylist() ([]PlaylistItem, error) {
	data, err := runCommand("get_property", "playlist")
	if err != nil {
		return nil, err
	}
	dataSlice, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected data type: %T", data)
	}

	var playlist []PlaylistItem
	for _, item := range dataSlice {
		itemMap, ok := item.(map[string]any)
		if ok {
			playlistItem := PlaylistItem{
				Filename: itemMap["filename"].(string),
				ID:       int(itemMap["id"].(float64)),
			}
			if title, ok := itemMap["title"].(string); ok {
				playlistItem.Title = title
			}
			if current, ok := itemMap["current"].(bool); ok && current {
				playlistItem.Current = true
			}
			if playing, ok := itemMap["playing"].(bool); ok && playing {
				playlistItem.Playing = true
			}
			playlist = append(playlist, playlistItem)
		}
	}
	return playlist, nil
}

func AddToPlaylist(u string) error {
	if u == "" {
		return fmt.Errorf("no episode URL provided")
	}
	if _, err := runCommand("get_property", "time-pos"); err != nil {
		cmd := exec.Command("/usr/bin/open", "iina://weblink?url="+url.QueryEscape(u))
		return cmd.Run()
	}
	playlist, _ := GetPlaylist()
	for _, item := range playlist {
		if item.Filename == u {
			return nil
		}
	}
	_, err := runCommand("loadfile", u, "append")
	return err
}

func PlayEpisode(url string, n ...bool) error {
	if url == "" {
		return fmt.Errorf("no episode URL provided")
	}
	next := len(n) > 0 && n[0]
	playlist, err := GetPlaylist()
	if err != nil && next {
		return err
	}
	var to int
	from := -1
	for idx, item := range playlist {
		if item.Current {
			to = idx
		}
		if to > 0 && item.Filename == url {
			from = idx
		}
	}
	if from == -1 {
		if err = AddToPlaylist(url); err != nil {
			return err
		}
		from = len(playlist)
	}
	if next {
		to++
	}
	if from != to {
		if _, err = runCommand("playlist-move", strconv.Itoa(from), strconv.Itoa(to)); err != nil {
			return err
		}
	}
	if !next {
		if _, err = runCommand("set_property", "playlist-pos", strconv.Itoa(to)); err != nil {
			return err
		}
		err = PlayPause(true)
	}
	return err
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

func RemoveFromPlaylist(i string) error {
	id, err := strconv.Atoi(i)
	if err != nil {
		return fmt.Errorf("invalid episode ID")
	}
	playlist, err := GetPlaylist()
	if err != nil {
		return err
	}
	for x, item := range playlist {
		if item.ID == id {
			if item.Current {
				if _, err = runCommand("playlist-next"); err != nil {
					return err
				}
			}
			_, err = runCommand("playlist-remove", strconv.Itoa(x))
			return err
		}
	}
	return fmt.Errorf("episode not found in playlist")
}

func PlayerControl(episodes []*Episode) *Item {
	playback := 0
	if p, err := runCommand("get_property", "playback-time"); err == nil {
		playback = int(p.(float64))
	}
	remain := episodes[0].Duration - playback
	totalRemain := -playback
	for _, e := range episodes {
		totalRemain += e.Duration
	}
	progressBar := ""
	for i := 0; i <= 50; i++ {
		if i == playback*50/episodes[0].Duration {
			progressBar += "✦"
		} else {
			progressBar += "·"
		}
	}
	title := fmt.Sprintf("%d Episode", len(episodes))
	if len(episodes) > 1 {
		title += "s"
	}
	title += fmt.Sprintf(", %s Remaining", formatDuration(totalRemain))
	item := Item{
		Title:    title,
		Subtitle: fmt.Sprintf("%s  %s  - %s", formatDuration(playback), progressBar, formatDuration(remain)),
		Icon:     &Icon{Path: "icons/playpause.png"},
	}
	item.SetVar("actionKeep", "playPause")

	cmd := &Mod{Subtitle: "Seek 30s backwards", Icon: &Icon{Path: "icons/rewind.png"}}
	cmd.SetVar("actionKeep", "30Back")
	item.Mods.Cmd = cmd

	alt := &Mod{Subtitle: "Play next episode", Icon: &Icon{Path: "icons/next.png"}}
	alt.SetVar("action", "next")
	item.Mods.Alt = alt

	shift := &Mod{Subtitle: "Save playlist", Icon: &Icon{Path: "icons/save.png"}}
	shift.SetVar("actionKeep", "save")
	item.Mods.Shift = shift

	return &item
}

func SavePlaylist() error {
	var playlist []string
	items, err := GetPlaylist()
	if err != nil {
		return err
	}
	for _, item := range items {
		if !item.Current && playlist == nil {
			continue
		}
		playlist = append(playlist, item.Filename)
	}
	if len(playlist) == 0 {
		return fmt.Errorf("no episodes in playlist")
	}
	file := fmt.Sprintf("%s/playlist.m3u", cacheDir)
	data := []byte(strings.Join(playlist, "\n"))
	return os.WriteFile(file, data, 0644)
}

func LoadPlaylist() error {
	file := fmt.Sprintf("%s/playlist.m3u", cacheDir)
	if _, err := os.Stat(file); err != nil {
		return err
	}
	if _, err := runCommand("loadlist", file); err != nil {
		cmd := exec.Command("/usr/bin/open", file)
		cmd.Run()
	}
	return nil
}
