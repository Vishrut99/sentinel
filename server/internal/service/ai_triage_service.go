package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultGeminiTriageURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"
	geminiAPIHostPrefix    = "https://generativelanguage.googleapis.com/"
)

var fallbackGeminiTriageURLs = []string{
	"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent",
	"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-001:generateContent",
}

type AITriageService interface {
	AnalyzeTicket(ctx context.Context, title, description string) (json.RawMessage, error)
}

type aiTriageService struct {
	url       string
	apiKey    string
	useGemini bool
	client    *http.Client
}

func NewAITriageService(url, apiKey string) AITriageService {
	trimmedURL := strings.TrimSpace(url)
	trimmedAPIKey := strings.TrimSpace(apiKey)

	if trimmedURL == "" && trimmedAPIKey == "" {
		return nil
	}
	if trimmedURL == "" && trimmedAPIKey != "" {
		trimmedURL = defaultGeminiTriageURL
	}

	return &aiTriageService{
		url:       trimmedURL,
		apiKey:    trimmedAPIKey,
		useGemini: isGeminiEndpoint(trimmedURL),
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *aiTriageService) AnalyzeTicket(ctx context.Context, title, description string) (json.RawMessage, error) {
	if s.useGemini {
		return s.analyzeWithGemini(ctx, title, description)
	}

	requestBody, err := json.Marshal(map[string]string{
		"title":       title,
		"description": description,
	})
	if err != nil {
		return nil, fmt.Errorf("AITriageService.AnalyzeTicket: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("AITriageService.AnalyzeTicket: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("AITriageService.AnalyzeTicket: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("AITriageService.AnalyzeTicket: %w", err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("AITriageService.AnalyzeTicket: unexpected status %d", response.StatusCode)
	}

	if !json.Valid(body) {
		return nil, fmt.Errorf("AITriageService.AnalyzeTicket: invalid JSON response")
	}

	var validateObject map[string]any
	if err := json.Unmarshal(body, &validateObject); err != nil {
		return nil, fmt.Errorf("AITriageService.AnalyzeTicket: %w", err)
	}

	return json.RawMessage(body), nil
}

func (s *aiTriageService) analyzeWithGemini(ctx context.Context, title, description string) (json.RawMessage, error) {
	prompt := fmt.Sprintf("Return only a valid JSON object with keys suggested_priority (low|medium|high|critical), suggested_category (string), sentiment (string), summary (string), required_skills (array of 1-5 concise skill names). Title: %s Description: %s", title, description)

	requestBody, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{{"text": prompt}},
			},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("AITriageService.analyzeWithGemini: %w", err)
	}

	candidateURLs := buildGeminiCandidateURLs(s.url)

	var lastErr error
	for _, endpoint := range candidateURLs {
		body, err := s.callGeminiEndpoint(ctx, endpoint, requestBody)
		if err != nil {
			lastErr = err
			continue
		}

		triageJSON, err := parseGeminiJSON(body)
		if err != nil {
			lastErr = err
			continue
		}

		return triageJSON, nil
	}

	return nil, fmt.Errorf("AITriageService.analyzeWithGemini: %w", lastErr)
}

func (s *aiTriageService) callGeminiEndpoint(ctx context.Context, endpoint string, requestBody []byte) ([]byte, error) {
	geminiURL := endpoint
	if s.apiKey != "" {
		parsedURL, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("AITriageService.callGeminiEndpoint: %w", err)
		}
		query := parsedURL.Query()
		query.Set("key", s.apiKey)
		parsedURL.RawQuery = query.Encode()
		geminiURL = parsedURL.String()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, geminiURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("AITriageService.callGeminiEndpoint: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("AITriageService.callGeminiEndpoint: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("AITriageService.callGeminiEndpoint: %w", err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("AITriageService.callGeminiEndpoint: %s", formatGeminiHTTPError(response.StatusCode, body))
	}

	return body, nil
}

func isGeminiEndpoint(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), geminiAPIHostPrefix)
}

func buildGeminiCandidateURLs(primary string) []string {
	candidates := make([]string, 0, 1+len(fallbackGeminiTriageURLs))
	appendUniqueURL := func(value string) {
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			return
		}
		for _, existing := range candidates {
			if existing == trimmedValue {
				return
			}
		}
		candidates = append(candidates, trimmedValue)
	}

	appendUniqueURL(primary)
	appendUniqueURL(defaultGeminiTriageURL)
	for _, endpoint := range fallbackGeminiTriageURLs {
		appendUniqueURL(endpoint)
	}

	return candidates
}

func formatGeminiHTTPError(statusCode int, body []byte) string {
	var responseBody struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &responseBody); err == nil && responseBody.Error.Message != "" {
		if responseBody.Error.Status != "" {
			return fmt.Sprintf("unexpected status %d (%s): %s", statusCode, responseBody.Error.Status, truncateForLog(responseBody.Error.Message, 180))
		}
		return fmt.Sprintf("unexpected status %d: %s", statusCode, truncateForLog(responseBody.Error.Message, 180))
	}

	return fmt.Sprintf("unexpected status %d body %s", statusCode, truncateForLog(strings.TrimSpace(string(body)), 180))
}

func parseGeminiJSON(body []byte) (json.RawMessage, error) {
	var geminiResponse struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &geminiResponse); err != nil {
		return nil, fmt.Errorf("AITriageService.parseGeminiJSON: %w", err)
	}

	if len(geminiResponse.Candidates) == 0 || len(geminiResponse.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("AITriageService.parseGeminiJSON: empty response")
	}

	rawText := geminiResponse.Candidates[0].Content.Parts[0].Text
	jsonText := extractJSONObject(rawText)
	if jsonText == "" || !json.Valid([]byte(jsonText)) {
		return nil, fmt.Errorf("AITriageService.parseGeminiJSON: invalid triage JSON")
	}

	return json.RawMessage(jsonText), nil
}

func truncateForLog(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "..."
}

func extractJSONObject(input string) string {
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")
	if start == -1 || end == -1 || end < start {
		return ""
	}
	return input[start : end+1]
}
