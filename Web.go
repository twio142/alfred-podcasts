package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
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
				Url  string `xml:"xmlUrl,attr"`
			} `xml:"outline"`
		} `xml:"outline"`
	} `xml:"body"`
}

type RSS struct {
	Channel struct {
		Items []ChannelItem `xml:"item"`
		Title string        `xml:"title"`
		Link  string        `xml:"link"`
		Desc  string        `xml:"description"`
		Image struct {
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

func RequestFeeds() ([]*Podcast, error) {
	opmlUrl := os.Getenv("FEEDS_URL")
	apiToken := os.Getenv("API_TOKEN")
	if opmlUrl == "" {
		return nil, fmt.Errorf("FEEDS_URL is required")
	}
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, _ := http.NewRequest("GET", opmlUrl, nil)
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Println("Error fetching OPML feeds:", err)
		return nil, err
	}
	defer resp.Body.Close()

	byteValue, _ := io.ReadAll(resp.Body)
	var opml Opml
	if err = xml.Unmarshal(byteValue, &opml); err != nil {
		log.Println("Error parsing XML:", err)
		return nil, err
	}

	var podcasts []*Podcast
	for _, outline := range opml.Body.Outlines.Feeds {
		podcast := Podcast{Name: outline.Text, URL: outline.Url}
		podcasts = append(podcasts, &podcast)
	}
	return podcasts, nil
}

func RequestRss(url string) (*RSS, error) {
	var client = &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(url)
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
