// Package chat stores conversations and messages in PocketBase and converts them to provider
// messages.
package chat

import (
	"encoding/json"
	"strings"

	"github.com/north-shore-software/kdnk/providers"
)

const (
	// PartTypeText is the MessagePart.Type of a plain-text part.
	PartTypeText   = "text"
	toolPartPrefix = "tool-"
)

// MessagePart is one part of a Message. Text parts have Type PartTypeText and set Text. Tool parts
// have Type "tool-" followed by the tool name, and carry a ToolPartData as JSON in Data.
type MessagePart struct {
	// Type is PartTypeText or "tool-<name>".
	Type string `json:"type"`
	// Text is the content of a text part.
	Text string `json:"text,omitempty"`
	// Data is the JSON-encoded ToolPartData of a tool part.
	Data json.RawMessage `json:"data,omitempty"`
}

// Message is a chat message as sent by the web client and stored in the CollectionMessage
// collection.
type Message struct {
	// ID identifies the message. Messages whose ID is already stored are not stored again.
	ID string `json:"id"`
	// Role is the sender, such as "user" or "assistant".
	Role string `json:"role"`
	// Parts is the message content, in order.
	Parts []MessagePart `json:"parts"`
}

// ToolPartData is the Data payload of a tool part: one tool call and its output.
type ToolPartData struct {
	// ToolCallID is the provider-assigned ID of the call.
	ToolCallID string `json:"toolCallId"`
	// ToolName is the name of the tool called.
	ToolName string `json:"toolName"`
	// Input is the JSON arguments the model passed.
	Input json.RawMessage `json:"input"`
	// Output is the string the tool returned.
	Output string `json:"output"`
	// ProviderData is the call's providers.ToolCall.ProviderData.
	ProviderData json.RawMessage `json:"providerData,omitempty"`
}

// TextPart returns a text MessagePart containing text.
func TextPart(text string) MessagePart {
	return MessagePart{Type: PartTypeText, Text: text}
}

// ToolPart returns a tool MessagePart for call and its output. It returns an error only if the
// data cannot be marshalled.
func ToolPart(call providers.ToolCall, output string) (MessagePart, error) {
	b, err := json.Marshal(ToolPartData{
		ToolCallID:   call.ID,
		ToolName:     call.Name,
		Input:        call.Args,
		Output:       output,
		ProviderData: call.ProviderData,
	})
	if err != nil {
		return MessagePart{}, err
	}
	return MessagePart{Type: toolPartPrefix + call.Name, Data: b}, nil
}

// ExtractNewMessages returns the messages in incoming whose ID does not appear in dbMsgs, in their
// original order.
func ExtractNewMessages(dbMsgs, incoming []Message) []Message {
	existingIDs := make(map[string]bool, len(dbMsgs))
	for _, m := range dbMsgs {
		existingIDs[m.ID] = true
	}

	var newMsgs []Message
	for _, m := range incoming {
		if !existingIDs[m.ID] {
			newMsgs = append(newMsgs, m)
		}
	}
	return newMsgs
}

// Hydrate converts stored messages to provider messages. Each message becomes one
// providers.Message with the same role, its text parts concatenated as Content and its tool parts
// as ToolCalls, omitted if it has neither. Each tool part's output follows it as a separate
// providers.Message with role "tool". Tool parts whose Data does not decode are skipped.
func Hydrate(msgs []Message) []providers.Message {
	var out []providers.Message
	for _, m := range msgs {
		msg := providers.Message{Role: m.Role}
		var results []providers.Message

		for _, p := range m.Parts {
			switch {
			case p.Type == PartTypeText:
				msg.Content += p.Text
			case strings.HasPrefix(p.Type, toolPartPrefix):
				var d ToolPartData
				if err := json.Unmarshal(p.Data, &d); err != nil {
					continue
				}
				msg.ToolCalls = append(msg.ToolCalls, providers.ToolCall{
					ID:           d.ToolCallID,
					Name:         d.ToolName,
					Args:         d.Input,
					ProviderData: d.ProviderData,
				})
				results = append(results, providers.Message{
					Role:       "tool",
					Content:    d.Output,
					ToolCallID: d.ToolCallID,
					ToolName:   d.ToolName,
				})
			}
		}

		if msg.Content != "" || len(msg.ToolCalls) > 0 {
			out = append(out, msg)
		}
		out = append(out, results...)
	}
	return out
}
