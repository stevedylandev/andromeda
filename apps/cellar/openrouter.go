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

type openrouterImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type openrouterContent struct {
	Type     string              `json:"type"`
	Text     string              `json:"text,omitempty"`
	ImageURL *openrouterImageURL `json:"image_url,omitempty"`
}

type openrouterMessage struct {
	Role    string              `json:"role"`
	Content []openrouterContent `json:"content"`
}

type openrouterResponseFormat struct {
	Type string `json:"type"`
}

type openrouterRequest struct {
	Model          string                    `json:"model"`
	MaxTokens      int                       `json:"max_tokens"`
	Messages       []openrouterMessage       `json:"messages"`
	ResponseFormat *openrouterResponseFormat `json:"response_format,omitempty"`
}

type openrouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"choices"`
}

const (
	openrouterModel  = "z-ai/glm-5.3-flash"
	openrouterPrompt = `Look at this wine bottle label. Return a JSON object with exactly these fields: {"name": "the full wine name", "origin": "region and/or country", "grape": "grape variety or blend", "background": "brief background about the wine and the winery, including any notable history or interesting facts"}. If you cannot determine a field, use an empty string. Respond with ONLY the JSON, no other text.`
)

func normalizeMediaType(mediaType string) string {
	if !strings.HasPrefix(mediaType, "image/") {
		return "image/" + mediaType
	}
	return mediaType
}

func analyzeWineImage(ctx context.Context, apiKey string, imageBytes []byte, mediaType string) (*AnalyzeResult, error) {
	dataURI := fmt.Sprintf("data:%s;base64,%s",
		normalizeMediaType(mediaType), base64.StdEncoding.EncodeToString(imageBytes))

	req := openrouterRequest{
		Model:     openrouterModel,
		MaxTokens: 1024,
		Messages: []openrouterMessage{{
			Role: "user",
			Content: []openrouterContent{
				{Type: "text", Text: openrouterPrompt},
				{Type: "image_url", ImageURL: &openrouterImageURL{URL: dataURI}},
			},
		}},
		ResponseFormat: &openrouterResponseFormat{Type: "json_object"},
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %s: %s", resp.Status, string(buf))
	}
	var cr openrouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("Failed to parse response: %w", err)
	}
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("No choices in response")
	}
	if cr.Choices[0].Error != nil {
		return nil, fmt.Errorf("API error %d: %s", cr.Choices[0].Error.Code, cr.Choices[0].Error.Message)
	}
	text := strings.TrimSpace(cr.Choices[0].Message.Content)
	if text == "" {
		return nil, fmt.Errorf("No text in response")
	}
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
