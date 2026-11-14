// Package agentsmith builds persistent, multi-user chat-agent apps from a single setup function.
package agentsmith

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/agent-smith/internal/chat"
	"github.com/north-shore-software/agent-smith/internal/plugin"
	"github.com/north-shore-software/agent-smith/internal/server"
	"github.com/north-shore-software/agent-smith/providers"
)

var errNotImplemented = errors.New("agentsmith: not implemented")

// Registry collects what an app defines during setup. It is frozen once setup returns.
type Registry struct {
	tools         []tool
	collections   []collection
	errs          []error
	provider      providers.Provider
	providerCalls int
	web           fs.FS
	webCalls      int
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

// Web sets the files served for GET requests that no other route matches, falling back to
// index.html for paths fsys does not contain. A nil fsys serves no web UI. Calling it more than
// once makes Run return an error.
func (r *Registry) Web(fsys fs.FS) {
	r.web = fsys
	r.webCalls++
}

// Run calls setup with a new Registry, then runs the PocketBase command line (serve, migrate and
// so on) for an app built from what setup registered. It returns setup's error; an error if
// Provider was not called, was called with nil, or was called more than once, if Web was called
// more than once, if a collection's type is invalid, if two collections or two tools share a name,
// or if a collection is named users, chat_conversation or chat_message, or starts with an
// underscore; or the command line's error.
//
// Collection tables are created or extended when the server starts serving, and the tools of
// every collection and every Tool are offered to the model. The system prompt is empty. The files
// passed to Web, if any, are served as the web UI.
func Run(setup func(*Registry) error) error {
	r := &Registry{}
	if err := setup(r); err != nil {
		return err
	}
	if err := r.validate(); err != nil {
		return err
	}
	app := server.New(server.Config{Plugin: registryPlugin{r}, Provider: r.provider, Web: r.web})
	for _, c := range r.collections {
		c.bind(app)
	}
	return app.Start()
}

func (r *Registry) validate() error {
	errs := r.errs
	if r.providerCalls > 1 {
		errs = append(errs, errors.New("agentsmith: Provider called more than once"))
	}
	if r.provider == nil {
		errs = append(errs, errors.New("agentsmith: no provider registered"))
	}
	if r.webCalls > 1 {
		errs = append(errs, errors.New("agentsmith: Web called more than once"))
	}
	reserved := []string{"users", chat.CollectionConversation, chat.CollectionMessage}
	seen := map[string]bool{}
	for _, c := range r.collections {
		name := c.collectionName()
		key := strings.ToLower(name)
		switch {
		case name == "" || strings.HasPrefix(name, "_"):
			errs = append(errs, fmt.Errorf("agentsmith: collection name %q is not allowed", name))
		case seen[key]:
			errs = append(errs, fmt.Errorf("agentsmith: collection %q registered more than once", name))
		}
		for _, rn := range reserved {
			if key == strings.ToLower(rn) {
				errs = append(errs, fmt.Errorf("agentsmith: collection name %q is reserved", name))
			}
		}
		seen[key] = true
	}
	tools := map[string]bool{}
	for _, t := range r.tools {
		if tools[t.def.Name] {
			errs = append(errs, fmt.Errorf("agentsmith: tool %q registered more than once", t.def.Name))
		}
		tools[t.def.Name] = true
	}
	return errors.Join(errs...)
}

type registryPlugin struct {
	r *Registry
}

func (registryPlugin) Name() string { return "agentsmith" }

func (p registryPlugin) Collections() []*core.Collection {
	cols := make([]*core.Collection, 0, len(p.r.collections))
	for _, c := range p.r.collections {
		cols = append(cols, c.pbCollection())
	}
	return cols
}

func (registryPlugin) SystemPrompt(context.Context, core.App, plugin.Session) (string, error) {
	return "", nil
}

func (p registryPlugin) Tools() []providers.Tool {
	defs := make([]providers.Tool, 0, len(p.r.tools))
	for _, t := range p.r.tools {
		defs = append(defs, t.def)
	}
	return defs
}

func (p registryPlugin) ExecuteTool(ctx context.Context, app core.App, s plugin.Session, name string, args json.RawMessage) (string, error) {
	for _, t := range p.r.tools {
		if t.def.Name != name {
			continue
		}
		out, err := t.run(ctx, app, Session{UserID: s.UserID, ConversationID: s.ConversationID}, args)
		if err != nil {
			return "", err
		}
		b, err := json.Marshal(out)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", fmt.Errorf("agentsmith: unknown tool %q", name)
}
