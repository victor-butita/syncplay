package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
)

type YouTubeOEmbedResponse struct {
	Title string `json:"title"`
}

func getYouTubeVideoInfo(videoID string) (string, error) {
	oEmbedURL := fmt.Sprintf("https://www.youtube.com/oembed?url=http://www.youtube.com/watch?v=%s&format=json", videoID)
	resp, err := http.Get(oEmbedURL)
	if err != nil {
		return "", fmt.Errorf("failed to make request to YouTube oEmbed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("YouTube oEmbed returned non-200 status: %s", resp.Status)
	}

	var oEmbedResp YouTubeOEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&oEmbedResp); err != nil {
		return "", fmt.Errorf("failed to decode YouTube oEmbed response: %w", err)
	}

	return oEmbedResp.Title, nil
}

func getYouTubeID(videoURL string) string {
	u, err := url.Parse(videoURL)
	if err != nil {
		return ""
	}
	if u.Host == "youtu.be" {
		return u.Path[1:]
	}
	if u.Host == "www.youtube.com" || u.Host == "youtube.com" {
		if u.Path == "/watch" {
			return u.Query().Get("v")
		}
		re := regexp.MustCompile(`/embed/([a-zA-Z0-9_-]{11})`)
		matches := re.FindStringSubmatch(u.Path)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}
