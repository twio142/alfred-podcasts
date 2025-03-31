package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/mozillazg/go-pinyin"
)

func matchString(text ...string) string {
	py := text
	for _, t := range text {
		for _, v := range pinyin.Convert(t, nil) {
			py = append(py, strings.Join(v, ""))
		}
	}
	return strings.Join(py, " ")
}

func formatDuration(duration int) string {
	if duration <= 0 {
		return "--:--"
	}
	hours := duration / 3600
	minutes := (duration % 3600) / 60
	seconds := duration % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	} else {
		return fmt.Sprintf("%02d:%02d", minutes, seconds)
	}
}

func downloadImage(url string, path string) {
	scpt := fmt.Sprintf("curl -m 10 -o '%s' '%s' && file --mime-type -b '%s' | grep -q '^image/' || rm -f '%s'", path, url, path, path)
	cmd := exec.Command("/bin/sh", "-c", scpt)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	cmd.Start()
}

func getCachePath(parts ...string) string {
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(part, "/", "%2F"), ":", "%3A")
	}
	return fmt.Sprintf("%s/%s", cacheDir, strings.Join(parts, "/"))
}

func readCache(path string, maxAge time.Duration, refreshTarget ...string) ([]byte, error) {
	// to read cache despite the maxAge, set maxAge to `time.Duration(math.MaxInt64)`
	if maxAge == 0 {
		return nil, fmt.Errorf("force cache refresh")
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("cache not found")
	} else if time.Since(info.ModTime()) > maxAge {
		refreshInBackground(refreshTarget)
	}
	return os.ReadFile(path)
}

func writeCache(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func clearOldCache() {
	scpt := fmt.Sprintf("find '%s' -type f -mtime +60 -delete", cacheDir+"/shownotes")
	cmd := exec.Command("/bin/sh", "-c", scpt)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	cmd.Start()
}

func getLockFile(refreshTarget []string) string {
	if refreshTarget[0] == "podcast" && len(refreshTarget) > 1 {
		lockfile := fmt.Sprintf("%s.lock", refreshTarget[1])
		return getCachePath("podcasts", lockfile)
	} else {
		lockfile := refreshTarget[0] + ".lock"
		return getCachePath(lockfile)
	}
}

func refreshInBackground(refreshTarget []string) {
	lockfile := getLockFile(refreshTarget)
	_, err := os.OpenFile(lockfile, os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		if os.IsExist(err) {
			return
		}
		log.Fatalf("Failed to create lock file: %v", err)
	}
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "refresh="+refreshTarget[0])
	if refreshTarget[0] == "podcast" && len(refreshTarget) > 1 {
		cmd.Env = append(cmd.Env, "podcastUuid="+refreshTarget[1])
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	cmd.Start()
}

func refreshCache(refreshTarget []string) error {
	defer os.Remove(getLockFile(refreshTarget))
	target := refreshTarget[0]
	switch target {
	case "podcast":
		if len(refreshTarget) < 2 {
			return fmt.Errorf("no podcast name provided")
		}
		p := &Podcast{UUID: refreshTarget[1]}
		return p.GetEpisodes(true)
	case "allPodcasts":
		clearOldCache()
		return GetAllPodcasts(true)
	case "up_next":
		_, err := GetUpNext(true)
		return err
	default:
		_, err := GetList(target, true)
		return err
	}
}

func Notify(message string, t ...string) {
	cmd := exec.Command("terminal-notifier")
	cmd.Args = append(cmd.Args, "-message", message, "-sender", "com.runningwithcrayons.Alfred", "-contentImage", "icon.png", "-title")
	if len(t) > 0 && t[0] != "" {
		cmd.Args = append(cmd.Args, t[0])
	} else {
		cmd.Args = append(cmd.Args, "Podcasts")
	}
	if err := cmd.Run(); err != nil {
		log.Println(err)
	}
}
