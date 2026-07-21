package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const geminiAPIBase = "https://generativelanguage.googleapis.com/v1beta/models"

// GeminiProvider talks to the Google Generative Language API directly over HTTP.
type GeminiProvider struct {
	httpClient *http.Client
}

func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{httpClient: &http.Client{}}
}

func (p *GeminiProvider) Name() string { return "gemini" }

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiRequestBody struct {
	Contents          []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

type geminiResponseBody struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int64 `json:"promptTokenCount"`
		CandidatesTokenCount int64 `json:"candidatesTokenCount"`
		TotalTokenCount      int64 `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// geminiRole maps our vendor-agnostic role to Gemini's expected role names.
// Gemini uses "user" and "model" (not "assistant").
func geminiRole(role string) string {
	if role == "assistant" {
		return "model"
	}
	return "user"
}

func toGeminiContents(msgs []Message) []geminiContent {
	var out []geminiContent
	for _, m := range msgs {
		if m.Role == "system" {
			continue
		}
		out = append(out, geminiContent{
			Role:  geminiRole(m.Role),
			Parts: []geminiPart{{Text: m.Content}},
		})
	}
	return out
}

func (p *GeminiProvider) buildBody(req CompletionRequest) geminiRequestBody {
	body := geminiRequestBody{
		Contents: toGeminiContents(req.Messages),
	}
	if req.System != "" {
		body.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.System}},
		}
	}
	if req.MaxTokens > 0 {
		body.GenerationConfig = geminiGenerationConfig{MaxOutputTokens: req.MaxTokens}
	}
	return body
}

func (p *GeminiProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResult, error) {
	body := p.buildBody(req)
	payload, err := json.Marshal(body)
	if err != nil {
		return CompletionResult{}, err
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiAPIBase, req.Model, req.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return CompletionResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return CompletionResult{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResult{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return CompletionResult{}, fmt.Errorf("gemini api error (%d): %s", resp.StatusCode, string(data))
	}

	var parsed geminiResponseBody
	if err := json.Unmarshal(data, &parsed); err != nil {
		return CompletionResult{}, err
	}

	var sb strings.Builder
	var finishReason string
	if len(parsed.Candidates) > 0 {
		finishReason = parsed.Candidates[0].FinishReason
		for _, part := range parsed.Candidates[0].Content.Parts {
			sb.WriteString(part.Text)
		}
	}

	return CompletionResult{
		Content:      sb.String(),
		TokensUsed:   parsed.UsageMetadata.TotalTokenCount,
		FinishReason: finishReason,
	}, nil
}

// Stream — Gemini supports server-streaming via `streamGenerateContent`, but
// to keep this implementation simple and dependency-free we implement Stream
// as a single Complete call followed by emitting the full content as one chunk.
// This is functionally correct (the UI still "streams" — just in one frame)
// and can be upgraded to true SSE streaming without changing the interface.
func (p *GeminiProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	out := make(chan StreamChunk, 2)

	go func() {
		defer close(out)
		result, err := p.Complete(ctx, req)
		if err != nil {
			out <- StreamChunk{Err: err, Done: true}
			return
		}
		out <- StreamChunk{Delta: result.Content}
		out <- StreamChunk{
			Done:         true,
			FinalContent: result.Content,
			TokensUsed:   result.TokensUsed,
		}
	}()

	return out, nil
}
