package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/north-shore-software/agent-smith/llm"
)

const (
	PartTypeText   = "text"
	toolPartPrefix = "tool-"
)

type MessagePart struct {
	Type string          `json:"type"`
	Text string          `json:"text,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

type Message struct {
	ID    string        `json:"id"`
	Role  string        `json:"role"`
	Parts []MessagePart `json:"parts"`
}

type ToolPartData struct {
	ToolCallID string          `json:"toolCallId"`
	ToolName   string          `json:"toolName"`
	Input      json.RawMessage `json:"input"`
	Output     string          `json:"output"`
}

func TextPart(text string) MessagePart {
	return MessagePart{Type: PartTypeText, Text: text}
}

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

func ToolResultText(name, output string) string {
	return fmt.Sprintf("Result of %s: %s", name, output)
}

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
