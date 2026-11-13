// Package chat stores conversations and messages in PocketBase and converts them to llm messages.
package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/north-shore-software/agent-smith/llm"
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
}

// TextPart returns a text MessagePart containing text.
func TextPart(text string) MessagePart {
	return MessagePart{Type: PartTypeText, Text: text}
}

// ToolPart returns a tool MessagePart for call and its output. It returns an error only if the
// data cannot be marshalled.
func ToolPart(call llm.ToolCall, output string) (MessagePart, error) {
	b, err := json.Marshal(ToolPartData{
		ToolCallID: call.ID,
		ToolName:   call.Name,
		Input:      call.Args,
		Output:     output,
	})
	if err != nil {
		return MessagePart{}, err
	}
	return MessagePart{Type: toolPartPrefix + call.Name, Data: b}, nil
}

// ToolResultText formats a tool's output as the text sent back to the model: "Result of <name>:
// <output>".
func ToolResultText(name, output string) string {
	return fmt.Sprintf("Result of %s: %s", name, output)
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

// Hydrate converts stored messages to llm messages. The text parts of each message are
// concatenated into one llm.Message with the same role, omitted if empty. Each tool part becomes a
// separate llm.Message with role "user" and ToolResultText content, placed after the text. Tool
// parts whose Data does not decode are skipped.
func Hydrate(msgs []Message) []llm.Message {
	var out []llm.Message
	for _, m := range msgs {
		var text string
		var results []llm.Message

		for _, p := range m.Parts {
			switch {
			case p.Type == PartTypeText:
				text += p.Text
			case strings.HasPrefix(p.Type, toolPartPrefix):
				var d ToolPartData
				if err := json.Unmarshal(p.Data, &d); err != nil {
					continue
				}
				results = append(results, llm.Message{
					Role:    "user",
					Content: ToolResultText(d.ToolName, d.Output),
				})
			}
		}

		if text != "" {
			out = append(out, llm.Message{Role: m.Role, Content: text})
		}
		out = append(out, results...)
	}
	return out
}
