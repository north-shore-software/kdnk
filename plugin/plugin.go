package plugin

import (
	"context"
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/agent-smith/llm"
)

type Session struct {
	ConversationID string
	UserID         string
}

type Plugin interface {
	Name() string
	Collections() []*core.Collection
	SystemPrompt(ctx context.Context, app core.App, s Session) (string, error)
	Tools() []llm.Tool
	ExecuteTool(ctx context.Context, app core.App, s Session, name string, args json.RawMessage) (string, error)
}
