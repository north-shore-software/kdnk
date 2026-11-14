package agentsmith

import (
	"context"
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/agent-smith/llm"
)

type tool struct {
	def llm.Tool
	run func(ctx context.Context, app core.App, s Session, args json.RawMessage) (any, error)
}

func (c *Store[T]) tools() []tool {
	switch c.restriction {
	case appendOnly:
		return []tool{c.createTool(), c.listTool(), c.getTool()}
	case readOnly:
		return []tool{c.listTool(), c.getTool()}
	}
	return []tool{c.createTool(), c.listTool(), c.getTool(), c.updateTool(), c.deleteTool()}
}

func (c *Store[T]) createTool() tool {
	return tool{
		def: llm.Tool{Name: c.name + "_create", Description: "Create a record in " + c.name + "."},
		run: func(ctx context.Context, app core.App, s Session, args json.RawMessage) (any, error) {
			return nil, errNotImplemented
		},
	}
}

func (c *Store[T]) listTool() tool {
	return tool{
		def: llm.Tool{Name: c.name + "_list", Description: "List records in " + c.name + ", newest first."},
		run: func(ctx context.Context, app core.App, s Session, args json.RawMessage) (any, error) {
			return nil, errNotImplemented
		},
	}
}

func (c *Store[T]) getTool() tool {
	return tool{
		def: llm.Tool{Name: c.name + "_get", Description: "Get a record in " + c.name + " by ID."},
		run: func(ctx context.Context, app core.App, s Session, args json.RawMessage) (any, error) {
			return nil, errNotImplemented
		},
	}
}

func (c *Store[T]) updateTool() tool {
	return tool{
		def: llm.Tool{Name: c.name + "_update", Description: "Update a record in " + c.name + " by ID."},
		run: func(ctx context.Context, app core.App, s Session, args json.RawMessage) (any, error) {
			return nil, errNotImplemented
		},
	}
}

func (c *Store[T]) deleteTool() tool {
	return tool{
		def: llm.Tool{Name: c.name + "_delete", Description: "Delete a record in " + c.name + " by ID."},
		run: func(ctx context.Context, app core.App, s Session, args json.RawMessage) (any, error) {
			return nil, errNotImplemented
		},
	}
}
