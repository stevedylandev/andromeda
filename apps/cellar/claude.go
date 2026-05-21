package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AnalyzeResult struct {
	Name       string `json:"name"`
	Origin     string `json:"origin"`
	Grape      string `json:"grape"`
	Background string `json:"background"`
}

type claudeImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type claudeContent struct {
	Type   string             `json:"type"`
	Source *claudeImageSource `json:"source,omitempty"`
	Text   string             `json:"text,omitempty"`
}

type claudeMessage struct {
	Role    string          `json:"role"`
	Content []claudeContent `json:"content"`
}

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

const claudeModel = "claude-sonnet-4-20250514"
const claudePrompt = `Look at this wine bottle label. Return a JSON object with exactly these fields: {"name": "the full wine name", "origin": "region and/or country", "grape": "grape variety or blend", "background": "brief background about the wine and the winery, including any notable history or interesting facts"}. If you cannot determine a field, use an empty string. Respond with ONLY the JSON, no other text.`

func analyzeWineImage(ctx context.Context, apiKey string, imageBytes []byte, mediaType string) (*AnalyzeResult, error) {
	encoded := base64.StdEncoding.EncodeToString(imageBytes)
	req := claudeRequest{
		Model:     claudeModel,
		MaxTokens: 1024,
		Messages: []claudeMessage{{
			Role: "user",
			Content: []claudeContent{
				{Type: "image", Source: &claudeImageSource{Type: "base64", MediaType: mediaType, Data: encoded}},
				{Type: "text", Text: claudePrompt},
			},
		}},
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %s: %s", resp.Status, string(buf))
	}
	var cr claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("Failed to parse response: %w", err)
	}
	var text string
	for _, c := range cr.Content {
		if c.Text != "" {
			text = c.Text
			break
		}
	}
	if text == "" {
		return nil, fmt.Errorf("No text in response")
	}
	text = strings.TrimSpace(text)
	if i := strings.Index(text, "{"); i >= 0 {
		if j := strings.LastIndex(text, "}"); j > i {
			text = text[i : j+1]
		}
	}
	var result AnalyzeResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("Failed to parse JSON: %w", err)
	}
	return &result, nil
}
