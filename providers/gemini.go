package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/north-shore-software/kdnk/internal/httpx"
)

const geminiBase = "https://generativelanguage.googleapis.com/v1beta"

// Gemini streams completions from the Gemini streamGenerateContent endpoint. It reads the API
// key from GEMINI_API_KEY on every call.
type Gemini struct {
	// Model is the Gemini model name, such as "gemini-3.6-flash". It is required.
	Model string
}

// ModelName returns p.Model.
func (p *Gemini) ModelName() string {
	return p.Model
}

func (*Gemini) sealed() {}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiFunctionCall struct {
	ID   string          `json:"id,omitempty"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args,omitempty"`
}

type geminiFunctionResponse struct {
	ID       string          `json:"id,omitempty"`
	Name     string          `json:"name"`
	Response json.RawMessage `json:"response"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"`
}

type geminiCallData struct {
	Provider         string `json:"provider"`
	ID               string `json:"id,omitempty"`
	ThoughtSignature string `json:"thoughtSignature,omitempty"`
}

func decodeGeminiCallData(raw json.RawMessage) geminiCallData {
	var d geminiCallData
	if len(raw) == 0 || json.Unmarshal(raw, &d) != nil || d.Provider != "gemini" {
		return geminiCallData{}
	}
	return d
}

func geminiResponse(output string) json.RawMessage {
	result := json.RawMessage(output)
	if !json.Valid(result) {
		result, _ = json.Marshal(output)
	}
	b, _ := json.Marshal(map[string]json.RawMessage{"result": result})
	return b
}

func orEmptyObject(args json.RawMessage) json.RawMessage {
	if len(args) == 0 {
		return json.RawMessage("{}")
	}
	return args
}

type geminiFunctionDeclaration struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations"`
}

type geminiRequest struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Tools             []geminiTool    `json:"tools,omitempty"`
}

type geminiStreamChunk struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// Stream implements Provider. System messages are joined into the system instruction, and
// the "assistant" role is sent as Gemini's "model" role, with its ToolCalls as functionCall parts.
// Consecutive "tool" messages are sent as one "user" turn of functionResponse parts, each with the
// response {"result": output}, where output is embedded as JSON if it is valid JSON and as a string
// otherwise. It returns an error if GEMINI_API_KEY or Model is empty, or if the request fails or
// returns a non-2xx status. Each tool call is emitted as an EventToolStart followed by an
// EventToolEnd with the full arguments, "{}" if Gemini sent none, and ProviderData holding
// Gemini's call ID and thought signature, which are sent back with the call. Usage has no
// TokensPerSecond.
func (p *Gemini) Stream(ctx context.Context, messages []Message, tools []Tool) (*Completion, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("gemini: GEMINI_API_KEY not set")
	}
	if p.Model == "" {
		return nil, fmt.Errorf("gemini: no model set")
	}

	contents := make([]geminiContent, 0, len(messages))
	var systemText strings.Builder
	geminiIDs := make(map[string]string)
	for _, m := range messages {
		switch m.Role {
		case "system":
			if systemText.Len() > 0 {
				systemText.WriteString("\n\n")
			}
			systemText.WriteString(m.Content)
			continue
		case "tool":
			part := geminiPart{FunctionResponse: &geminiFunctionResponse{
				ID:       geminiIDs[m.ToolCallID],
				Name:     m.ToolName,
				Response: geminiResponse(m.Content),
			}}
			if n := len(contents); n > 0 && contents[n-1].Role == "user" && contents[n-1].Parts[0].FunctionResponse != nil {
				contents[n-1].Parts = append(contents[n-1].Parts, part)
			} else {
				contents = append(contents, geminiContent{Role: "user", Parts: []geminiPart{part}})
			}
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		var parts []geminiPart
		if m.Content != "" || len(m.ToolCalls) == 0 {
			parts = append(parts, geminiPart{Text: m.Content})
		}
		for _, tc := range m.ToolCalls {
			d := decodeGeminiCallData(tc.ProviderData)
			if d.ID != "" {
				geminiIDs[tc.ID] = d.ID
			}
			parts = append(parts, geminiPart{
				FunctionCall:     &geminiFunctionCall{ID: d.ID, Name: tc.Name, Args: orEmptyObject(tc.Args)},
				ThoughtSignature: d.ThoughtSignature,
			})
		}
		contents = append(contents, geminiContent{Role: role, Parts: parts})
	}

	var systemInstruction *geminiContent
	if systemText.Len() > 0 {
		systemInstruction = &geminiContent{Parts: []geminiPart{{Text: systemText.String()}}}
	}

	var gTools []geminiTool
	if len(tools) > 0 {
		decls := make([]geminiFunctionDeclaration, 0, len(tools))
		for _, t := range tools {
			decls = append(decls, geminiFunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			})
		}
		gTools = []geminiTool{{FunctionDeclarations: decls}}
	}

	body, err := json.Marshal(geminiRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
		Tools:             gTools,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", geminiBase, p.ModelName())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := httpx.Streaming().Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	ch := make(chan StreamEvent)
	done := make(chan struct{})
	var finalUsage *Usage
	go func() {
		defer close(done)
		defer close(ch)
		defer resp.Body.Close()

		usage := Usage{Provider: "gemini", Model: p.ModelName()}
		var sawUsage bool

		var calls int

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}
			var chunk geminiStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			// usageMetadata is cumulative and present on later chunks; keep the latest.
			if u := chunk.UsageMetadata; u != nil {
				usage.PromptTokens = u.PromptTokenCount
				usage.CompletionTokens = u.CandidatesTokenCount
				usage.TotalTokens = u.TotalTokenCount
				sawUsage = true
			}
			if len(chunk.Candidates) == 0 {
				continue
			}
			for _, part := range chunk.Candidates[0].Content.Parts {
				if part.Text != "" {
					select {
					case ch <- StreamEvent{Kind: EventText, Text: part.Text}:
					case <-ctx.Done():
						return
					}
				}
				if fc := part.FunctionCall; fc != nil && fc.Name != "" {
					id := fc.ID
					if id == "" {
						id = fmt.Sprintf("call-%d-%d", time.Now().UnixNano(), calls)
					}
					calls++
					var data json.RawMessage
					if fc.ID != "" || part.ThoughtSignature != "" {
						data, _ = json.Marshal(geminiCallData{Provider: "gemini", ID: fc.ID, ThoughtSignature: part.ThoughtSignature})
					}
					for _, ev := range []StreamEvent{
						{Kind: EventToolStart, ToolCallID: id, ToolName: fc.Name},
						{Kind: EventToolEnd, ToolCallID: id, ToolName: fc.Name, Args: orEmptyObject(fc.Args), ProviderData: data},
					} {
						select {
						case ch <- ev:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
		if sawUsage {
			finalUsage = &usage
		}
	}()

	return &Completion{
		Events: ch,
		Wait:   func() *Usage { <-done; return finalUsage },
	}, nil
}
