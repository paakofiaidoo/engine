package ai

import "context"

// Provider is the interface every AI backend must implement.
// Implementations use plain net/http against each vendor's REST API —
// no heavyweight SDKs required, keeps the engine binary small and the
// build dependency-free (important for cross-compilation in goreleaser).
type Provider interface {
	// Name returns the provider identifier, e.g. "claude", "gemini".
	Name() string

	// Complete sends a prompt (with full message history) and returns
	// the complete response plus the number of tokens consumed.
	Complete(ctx context.Context, req CompletionRequest) (CompletionResult, error)

	// Stream sends a prompt and streams back response chunks via the
	// returned channel. The channel is closed when the response completes
	// or an error occurs (in which case `err` receives the error).
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}

// Message is a single turn in the conversation, vendor-agnostic.
type Message struct {
	Role    string // "user" | "assistant" | "system"
	Content string
}

// CompletionRequest is the vendor-agnostic request shape.
type CompletionRequest struct {
	APIKey      string
	Model       string
	System      string // system prompt (project context, brief, etc.)
	Messages    []Message
	MaxTokens   int
}

// CompletionResult is the vendor-agnostic response shape.
type CompletionResult struct {
	Content      string
	TokensUsed   int64
	FinishReason string
}

// StreamChunk is a single piece of a streamed response.
type StreamChunk struct {
	Delta        string
	Done         bool
	FinalContent string // populated on the final chunk
	TokensUsed   int64  // populated on the final chunk
	Err          error
}
