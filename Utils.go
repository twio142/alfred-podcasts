package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
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

func calculateDuration(durationStr string) int {
	num, err := strconv.Atoi(durationStr)
	if err == nil {
		return num
	}
	parts := strings.Split(durationStr, ":")
	if len(parts) == 3 {
		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		seconds, _ := strconv.Atoi(parts[2])
		return (hours * 3600) + (minutes * 60) + seconds
	} else if len(parts) == 2 {
		minutes, _ := strconv.Atoi(parts[0])
		seconds, _ := strconv.Atoi(parts[1])
		return (minutes * 60) + seconds
	} else {
		seconds, _ := strconv.Atoi(parts[0])
		return seconds
	}
}

func downloadImage(url string, path string) {
	scpt := fmt.Sprintf("curl -m 10 -o '%s' '%s' && file --mime-type -b '%s' | grep -q '^image/' && sips -Z 256 '%s' || rm -f '%s'", path, url, path, path, path)
	cmd := exec.Command("/bin/sh", "-c", scpt)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	cmd.Start()
}

func longestString(str ...string) string {
	var longest string
	for _, s := range str {
		s = strings.TrimSpace(s)
		if len(s) > len(longest) {
			longest = s
		}
	}
	return longest
}

func parseDate(dateStr string) time.Time {
	formats := []string{
		"Mon, 2 Jan 2006 15:04:05 MST",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 -0700",
	}

	var t time.Time
	var err error
	for _, format := range formats {
		t, err = time.Parse(format, dateStr)
		if err == nil {
			return t
		}
	}

	return t.In(time.Local)
}

func getCachePath(parts ...string) string {
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(part, "/", "%2F"), ":", "%3A")
	}
	return fmt.Sprintf("%s/%s", cacheDir, strings.Join(parts, "/"))
}

func readCache(path string, maxAge time.Duration) ([]byte, error) {
	// to read cache despite the maxAge, set maxAge to `time.Duration(math.MaxInt64)`
	if maxAge == 0 {
		return nil, fmt.Errorf("force cache refresh")
	}
	if info, err := os.Stat(path); os.IsNotExist(err) || time.Since(info.ModTime()) > maxAge {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache not found")
		} else {
			refreshInBackground()
		}
	}
	return os.ReadFile(path)
}

func writeCache(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func clearOldCache() {
	cmd := exec.Command("find", cacheDir+"/shownotes", "-type", "f", "-mtime", "+30", "-exec", "rm", "{}", ";")
	cmd.Run()
}

func refreshInBackground() {
	_, err := os.OpenFile(getCachePath("podcasts.lock"), os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		if os.IsExist(err) {
			return
		}
		log.Fatalf("Failed to create lock file: %v", err)
	}
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "action=refreshInBackground")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	cmd.Start()
}

func Notify(message string, t ...string) {
	cmd := exec.Command("terminal-notifier")
	cmd.Args = append(cmd.Args, "-message", message, "-sender", "com.runningwithcrayons.Alfred", "-contentImage", "icons/podcast.png", "-title")
	if len(t) > 0 && t[0] != "" {
		cmd.Args = append(cmd.Args, t[0])
	} else {
		cmd.Args = append(cmd.Args, "Podcasts")
	}
	if err := cmd.Run(); err != nil {
		log.Println(err)
	}
}
