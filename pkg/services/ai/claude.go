package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const claudeAPIURL = "https://api.anthropic.com/v1/messages"
const claudeAPIVersion = "2023-06-01"

// ClaudeProvider talks to the Anthropic Messages API directly over HTTP.
type ClaudeProvider struct {
	httpClient *http.Client
}

func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{httpClient: &http.Client{}}
}

func (p *ClaudeProvider) Name() string { return "claude" }

type claudeRequestBody struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []claudeMessage `json:"messages"`
	Stream    bool            `json:"stream,omitempty"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponseBody struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int64 `json:"input_tokens"`
		OutputTokens int64 `json:"output_tokens"`
	} `json:"usage"`
}

func toClaudeMessages(msgs []Message) []claudeMessage {
	var out []claudeMessage
	for _, m := range msgs {
		role := m.Role
		if role == "system" {
			continue // system is sent separately
		}
		out = append(out, claudeMessage{Role: role, Content: m.Content})
	}
	return out
}

func (p *ClaudeProvider) buildRequest(ctx context.Context, req CompletionRequest, stream bool) (*http.Request, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	body := claudeRequestBody{
		Model:     req.Model,
		MaxTokens: maxTokens,
		System:    req.System,
		Messages:  toClaudeMessages(req.Messages),
		Stream:    stream,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeAPIURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", req.APIKey)
	httpReq.Header.Set("anthropic-version", claudeAPIVersion)
	return httpReq, nil
}

func (p *ClaudeProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResult, error) {
	httpReq, err := p.buildRequest(ctx, req, false)
	if err != nil {
		return CompletionResult{}, err
	}

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
		return CompletionResult{}, fmt.Errorf("claude api error (%d): %s", resp.StatusCode, string(data))
	}

	var parsed claudeResponseBody
	if err := json.Unmarshal(data, &parsed); err != nil {
		return CompletionResult{}, err
	}

	var sb strings.Builder
	for _, c := range parsed.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}

	return CompletionResult{
		Content:      sb.String(),
		TokensUsed:   parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		FinishReason: parsed.StopReason,
	}, nil
}

// claudeStreamEvent mirrors the relevant subset of Anthropic's SSE stream format.
type claudeStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int64 `json:"output_tokens"`
	} `json:"usage"`
}

func (p *ClaudeProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	httpReq, err := p.buildRequest(ctx, req, true)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("claude api error (%d): %s", resp.StatusCode, string(data))
	}

	out := make(chan StreamChunk, 32)

	go func() {
		defer resp.Body.Close()
		defer close(out)

		var full strings.Builder
		var totalTokens int64

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				break
			}

			var evt claudeStreamEvent
			if err := json.Unmarshal([]byte(payload), &evt); err != nil {
				continue
			}

			switch evt.Type {
			case "content_block_delta":
				if evt.Delta.Type == "text_delta" && evt.Delta.Text != "" {
					full.WriteString(evt.Delta.Text)
					select {
					case out <- StreamChunk{Delta: evt.Delta.Text}:
					case <-ctx.Done():
						return
					}
				}
			case "message_delta":
				if evt.Usage.OutputTokens > 0 {
					totalTokens = evt.Usage.OutputTokens
				}
			}
		}

		out <- StreamChunk{
			Done:         true,
			FinalContent: full.String(),
			TokensUsed:   totalTokens,
		}
	}()

	return out, nil
}
