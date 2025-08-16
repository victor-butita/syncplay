package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Structs for manually creating the JSON request body
type GeminiRequest struct {
	Contents []*Content `json:"contents"`
}
type Content struct {
	Parts []*Part `json:"parts"`
}
type Part struct {
	Text string `json:"text"`
}

// Structs for manually parsing the JSON response body
type GeminiResponse struct {
	Candidates []*Candidate `json:"candidates"`
}
type Candidate struct {
	Content *Content `json:"content"`
}

// This function now uses a direct HTTP call, bypassing the broken library.
func generateIcebreakers(videoTitle string) ([]string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set in .env file")
	}

	// The stable v1 API endpoint, not v1beta
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-latest:generateContent?key=" + apiKey

	prompt := fmt.Sprintf("Based on the YouTube video title '%s', generate exactly 3 short, fun, and engaging conversation starters or 'icebreakers' for a watch party. Format them as a numbered list, like '1. Question one?'. Do not add any extra introduction or conclusion.", videoTitle)

	// Manually construct the request body
	reqBody := GeminiRequest{
		Contents: []*Content{
			{
				Parts: []*Part{
					{
						Text: prompt,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	// Manually parse the response
	var apiResponse GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Safely extract the text from the response structure
	if len(apiResponse.Candidates) > 0 &&
		apiResponse.Candidates[0].Content != nil &&
		len(apiResponse.Candidates[0].Content.Parts) > 0 {
		rawText := apiResponse.Candidates[0].Content.Parts[0].Text
		return parseIcebreakers(rawText), nil
	}

	return nil, fmt.Errorf("unexpected API response format")
}

// This helper function remains the same
func parseIcebreakers(raw string) []string {
	var icebreakers []string
	re := regexp.MustCompile(`\d+\.\s*(.+)`)
	matches := re.FindAllStringSubmatch(raw, -1)
	for _, match := range matches {
		if len(match) > 1 {
			icebreakers = append(icebreakers, strings.TrimSpace(match[1]))
		}
	}
	if len(icebreakers) == 0 {
		nonEmptyLines := []string{}
		for _, line := range strings.Split(raw, "\n") {
			if strings.TrimSpace(line) != "" {
				nonEmptyLines = append(nonEmptyLines, line)
			}
		}
		return nonEmptyLines
	}
	return icebreakers
}
