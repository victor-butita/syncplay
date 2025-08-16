package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func generateIcebreakers(videoTitle string) ([]string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-pro")
	prompt := fmt.Sprintf("Based on the YouTube video title '%s', generate exactly 3 short, fun, and engaging conversation starters or 'icebreakers' for a watch party. Format them as a numbered list, like '1. Question one?'. Do not add any extra introduction or conclusion.", videoTitle)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from AI model")
	}

	content := resp.Candidates[0].Content.Parts[0].(genai.Text)
	return parseIcebreakers(string(content)), nil
}

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
