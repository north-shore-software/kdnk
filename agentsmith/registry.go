// Package agentsmith builds persistent, multi-user chat-agent apps from a single setup function.
package agentsmith

import "errors"

var errNotImplemented = errors.New("agentsmith: not implemented")

// Registry collects what an app defines during setup. It is frozen once setup returns.
type Registry struct {
	tools []tool
}

// Session identifies the user and conversation a request belongs to.
type Session struct {
	// UserID is the ID of the authenticated user.
	UserID string
	// ConversationID is the ID of the conversation.
	ConversationID string
}

// Run calls setup with a new Registry and returns the error setup returns. Serving is not
// implemented yet: if setup succeeds, Run returns an error.
func Run(setup func(*Registry) error) error {
	if err := setup(&Registry{}); err != nil {
		return err
	}
	return errNotImplemented
}
