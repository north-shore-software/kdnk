// Package llm defines the provider-neutral types for streaming chat completions with tool calls,
// and the process-wide registry that maps model sets and roles to models and providers.
package llm

import (
	"context"
	"encoding/json"
)

// Message is one message sent to a Provider.
type Message struct {
	// Role is the sender. The server uses "system", "user" and "assistant".
	Role string `json:"role"`
	// Content is the message text.
	Content string `json:"content"`
}

// Tool describes a function the model may call.
type Tool struct {
	// Name is the name the model calls the tool by.
	Name string `json:"name"`
	// Description tells the model what the tool does.
	Description string `json:"description"`
	// Parameters is a JSON Schema object describing the arguments.
	Parameters json.RawMessage `json:"parameters"`
}

// ToolCall is a model's request to call a Tool.
type ToolCall struct {
	// ID is assigned by the Provider and identifies the call within the stream.
	ID string `json:"id"`
	// Name is the name of the tool called.
	Name string `json:"name"`
	// Args is the JSON arguments.
	Args json.RawMessage `json:"args"`
}

// StreamEventKind identifies the kind of a StreamEvent.
type StreamEventKind int

const (
	// EventText carries a chunk of response text in Text.
	EventText StreamEventKind = iota
	// EventToolStart announces a tool call, with ToolCallID and ToolName set.
	EventToolStart
	// EventToolArgDelta carries a chunk of a tool call's argument JSON in Text. The bundled
	// providers do not emit it.
	EventToolArgDelta
	// EventToolEnd completes a tool call, with ToolCallID, ToolName and the full Args set.
	EventToolEnd
)

// StreamEvent is one event in a Completion's stream. Which fields are set depends on Kind.
type StreamEvent struct {
	// Kind is the kind of event.
	Kind StreamEventKind
	// Text is the text or argument chunk.
	Text string
	// ToolCallID identifies the tool call.
	ToolCallID string
	// ToolName is the name of the tool called.
	ToolName string
	// Args is the complete JSON arguments of the tool call.
	Args json.RawMessage
}

// Role names the job a model is used for. The active ModelSet maps each Role to a model.
type Role string

const (
	// RoleChat is the model used to answer chat requests.
	RoleChat Role = "chat"
)

// Usage reports token counts for a completed stream.
type Usage struct {
	// Provider is the provider name: "ollama" or "gemini" for the bundled providers.
	Provider string
	// Model is the model name.
	Model string
	// PromptTokens is the number of input tokens.
	PromptTokens int
	// CompletionTokens is the number of generated tokens.
	CompletionTokens int
	// TotalTokens is the total number of tokens.
	TotalTokens int
	// TokensPerSecond is the generation rate, or zero if the provider does not report it.
	TokensPerSecond float64
}

// Completion is a streamed response in progress.
type Completion struct {
	// Events delivers the stream and is closed when it ends.
	Events <-chan StreamEvent
	// Wait blocks until the stream has ended and returns its Usage, or nil if the provider
	// reported none.
	Wait func() *Usage
}

// Provider streams model responses.
type Provider interface {
	// Stream sends messages and the available tools to the model and returns the response stream.
	// An error means the stream could not be started.
	Stream(ctx context.Context, messages []Message, tools []Tool) (*Completion, error)
}
