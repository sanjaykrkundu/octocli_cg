package llm

import "context"

// Message is a chat message sent to or received from an LLM.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest captures provider-independent chat parameters.
type ChatRequest struct {
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
}

// StreamFunc receives response deltas as they arrive.
type StreamFunc func(delta string) error

// Client is the provider abstraction used by the agent loop.
type Client interface {
	StreamChat(ctx context.Context, request ChatRequest, onDelta StreamFunc) error
}
