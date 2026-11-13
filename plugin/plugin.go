// Package plugin defines the interface an application implements to give the server its prompt,
// tools and data.
package plugin

import (
	"context"
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/agent-smith/llm"
)

// Session identifies the conversation and user a Plugin call is made for.
type Session struct {
	// ConversationID is the conversation ID the client sent with the chat request.
	ConversationID string
	// UserID is the ID of the authenticated user's record in the users collection.
	UserID string
}

// Plugin is an application built on agent-smith. server.New serves exactly one.
type Plugin interface {
	// Name identifies the plugin in server logs.
	Name() string
	// Collections returns the PocketBase collections the plugin needs. When the server starts
	// serving, each one whose name does not exist yet is created. Existing collections are left
	// unchanged.
	Collections() []*core.Collection
	// SystemPrompt returns the system prompt for a chat request. It is called once per request,
	// before the first model call. An empty string sends no system message. An error fails the
	// request.
	SystemPrompt(ctx context.Context, app core.App, s Session) (string, error)
	// Tools returns the tools offered to the model. It is called once per chat request.
	Tools() []llm.Tool
	// ExecuteTool runs the tool called name with the model's JSON arguments, and returns its
	// output, which is sent to the client and back to the model. If it returns an error,
	// {"error":"<message>"} is sent instead.
	ExecuteTool(ctx context.Context, app core.App, s Session, name string, args json.RawMessage) (string, error)
}
