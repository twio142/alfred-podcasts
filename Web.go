package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

type Opml struct {
	Body struct {
		Outlines struct {
			Feeds []struct {
				Text string `xml:"text,attr"`
				URL  string `xml:"xmlUrl,attr"`
			} `xml:"outline"`
		} `xml:"outline"`
	} `xml:"body"`
}

type RSS struct {
	Channel struct {
		Items  []ChannelItem `xml:"item"`
		Title  string        `xml:"title"`
		Link   string        `xml:"link"`
		Desc   string        `xml:"description"`
		Author string        `xml:"author"`
		Image  struct {
			Href string `xml:"href,attr"`
			URL  string `xml:"url"`
		} `xml:"image"`
		Summary string `xml:"summary"`
	} `xml:"channel"`
}

type ChannelItem struct {
	Title     string `xml:"title"`
	Desc      string `xml:"description"`
	Content   string `xml:"content:encoded"`
	Date      string `xml:"pubDate"`
	Enclosure struct {
		URL string `xml:"url,attr"`
	} `xml:"enclosure"`
	Link     string `xml:"link"`
	Duration string `xml:"duration"`
	Image    struct {
		Href string `xml:"href,attr"`
		URL  string `xml:"url"`
	} `xml:"image"`
	Summary string `xml:"summary"`
}

func RequestOpml() (*Opml, error) {
	opmlURL := os.Getenv("FEEDS_URL")
	apiToken := os.Getenv("API_TOKEN")
	if opmlURL == "" {
		return nil, fmt.Errorf("FEEDS_URL is required")
	}
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, _ := http.NewRequest("GET", opmlURL, nil)
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Println("Error fetching OPML feeds:", err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP request failed with status %d", resp.StatusCode)
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	byteValue, _ := io.ReadAll(resp.Body)
	var opml Opml
	if err = xml.Unmarshal(byteValue, &opml); err != nil {
		log.Println("Error parsing XML:", err)
		return nil, err
	}
	return &opml, nil
}

func RequestFeeds() ([]*Podcast, error) {
	opml, err := RequestOpml()
	if err != nil {
		return nil, err
	}
	var podcasts []*Podcast
	for _, outline := range opml.Body.Outlines.Feeds {
		podcast := Podcast{Name: outline.Text, URL: outline.URL}
		podcasts = append(podcasts, &podcast)
	}
	if files, err := os.ReadDir(getCachePath("podcasts")); err == nil {
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			decodedName, _ := url.PathUnescape(file.Name())
			found := false
			for _, podcast := range podcasts {
				if podcast.Name == decodedName {
					found = true
					break
				}
			}
			if !found {
				os.Remove(getCachePath("podcasts", file.Name()))
				os.Remove(getCachePath("artworks", file.Name()))
			}
		}
	}
	return podcasts, nil
}

func SubscribeNewFeed(podcast *Podcast) (*Podcast, error) {
	u, err := url.Parse(podcast.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL")
	}
	podcast.URL = strings.TrimSpace(u.String())
	if err = podcast.GetEpisodes(true); err != nil {
		return nil, err
	}
	opml, err := RequestOpml()
	if err != nil {
		return nil, err
	}
	opml.Body.Outlines.Feeds = append(opml.Body.Outlines.Feeds, struct {
		Text string `xml:"text,attr"`
		URL  string `xml:"xmlUrl,attr"`
	}{Text: podcast.Name, URL: podcast.URL})
	xmlData, err := xml.MarshalIndent(opml, "", "    ")
	if err != nil {
		return nil, err
	}
	return podcast, UpdateFileAndCommit(string(xmlData))
}

func UnsubscribeFeed(podcast *Podcast) (*Podcast, error) {
	opml, err := RequestOpml()
	if err != nil {
		return nil, err
	}
	var feeds []struct {
		Text string `xml:"text,attr"`
		URL  string `xml:"xmlUrl,attr"`
	}
	for _, feed := range opml.Body.Outlines.Feeds {
		if feed.URL == podcast.URL {
			podcast.Name = feed.Text
		} else {
			feeds = append(feeds, feed)
		}
	}
	if len(feeds) == len(opml.Body.Outlines.Feeds) {
		return nil, fmt.Errorf("feed not found")
	}
	opml.Body.Outlines.Feeds = feeds
	xmlData, err := xml.MarshalIndent(opml, "", "    ")
	if err != nil {
		return nil, err
	}
	return podcast, UpdateFileAndCommit(string(xmlData))
}

func RequestRss(url string) (*RSS, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; rv:45.0) Gecko/20100101 Firefox/45.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	byteValue, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bodyString := string(byteValue)
	bodyString = strings.ReplaceAll(bodyString, "&bull;", "•")
	reAmp := regexp.MustCompile(`&(?:(#?[xX]?(?:[0-9a-fA-F]+|\w{1,8}));)?`)
	bodyString = reAmp.ReplaceAllStringFunc(bodyString, func(m string) string {
		if m == "&" {
			return "&amp;"
		}
		return m
	})
	var rss RSS
	if err = xml.Unmarshal([]byte(bodyString), &rss); err != nil {
		return nil, err
	}
	return &rss, nil
}

func (r *RSS) desc() string {
	desc := longestString(r.Channel.Desc, r.Channel.Summary)
	re := regexp.MustCompile(`<(br|hr)[^>]*>`)
	desc = re.ReplaceAllString(desc, "\n")
	re = regexp.MustCompile(`</?(p|span|strong|b|em|i)[^>]*>`)
	desc = re.ReplaceAllString(desc, "")
	re = regexp.MustCompile(`<a [^>]*>([^<]+)</a>`)
	desc = re.ReplaceAllString(desc, "$1")
	desc = strings.TrimSpace(desc)
	return desc
}
