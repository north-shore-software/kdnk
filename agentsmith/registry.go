// Package agentsmith builds persistent, multi-user chat-agent apps from a single setup function.
package agentsmith

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/agent-smith/internal/plugin"
	"github.com/north-shore-software/agent-smith/internal/server"
	"github.com/north-shore-software/agent-smith/providers"
)

var errNotImplemented = errors.New("agentsmith: not implemented")

// Registry collects what an app defines during setup. It is frozen once setup returns.
type Registry struct {
	tools         []tool
	provider      providers.Provider
	providerCalls int
}

// Session identifies the user and conversation a request belongs to.
type Session struct {
	// UserID is the ID of the authenticated user.
	UserID string
	// ConversationID is the ID of the conversation.
	ConversationID string
}

// Provider sets the provider that answers every chat request. Calling it more than once makes
// Run return an error.
func (r *Registry) Provider(p providers.Provider) {
	r.provider = p
	r.providerCalls++
}

// Run calls setup with a new Registry, then runs the PocketBase command line (serve, migrate and
// so on) for an app built from what setup registered. It returns setup's error; an error if
// Provider was not called, was called with nil, or was called more than once; or the command
// line's error. Registered tools and collections are not offered to the model, the system
// prompt is empty, and no web UI is served.
func Run(setup func(*Registry) error) error {
	r := &Registry{}
	if err := setup(r); err != nil {
		return err
	}
	if r.providerCalls > 1 {
		return errors.New("agentsmith: Provider called more than once")
	}
	if r.provider == nil {
		return errors.New("agentsmith: no provider registered")
	}
	return server.New(server.Config{Plugin: registryPlugin{}, Provider: r.provider}).Start()
}

type registryPlugin struct{}

func (registryPlugin) Name() string { return "agentsmith" }

func (registryPlugin) Collections() []*core.Collection { return nil }

func (registryPlugin) SystemPrompt(context.Context, core.App, plugin.Session) (string, error) {
	return "", nil
}

func (registryPlugin) Tools() []providers.Tool { return nil }

func (registryPlugin) ExecuteTool(context.Context, core.App, plugin.Session, string, json.RawMessage) (string, error) {
	return "", errNotImplemented
}
