package agentsmith

import (
	"context"
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/agent-smith/providers"
)

// Tool adds a custom tool named name, described to the model by description, to r. It is not
// implemented yet: the tool has no JSON Schema, and calling it returns an error without running
// handler.
func Tool[In any](r *Registry, name, description string, handler func(ctx context.Context, app core.App, s Session, in In) (any, error)) {
	r.tools = append(r.tools, tool{
		def: providers.Tool{Name: name, Description: description},
		run: func(ctx context.Context, app core.App, s Session, args json.RawMessage) (any, error) {
			return nil, errNotImplemented
		},
	})
}
