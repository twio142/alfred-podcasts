package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
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
