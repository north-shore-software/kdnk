package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/north-shore-software/agent-smith/llm"
)

type streamWriter struct {
	w        http.ResponseWriter
	flusher  http.Flusher
	canFlush bool
	started  bool

	genStart   time.Time
	lastEmit   time.Time
	tokenCount int
}

func newStreamWriter(w http.ResponseWriter) *streamWriter {
	flusher, canFlush := w.(http.Flusher)
	return &streamWriter{w: w, flusher: flusher, canFlush: canFlush}
}

func (s *streamWriter) send(event any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(s.w, "data: %s\n\n", data)
	if s.canFlush {
		s.flusher.Flush()
	}
}

func (s *streamWriter) start() {
	s.w.Header().Set("Content-Type", "text/event-stream")
	s.w.Header().Set("x-vercel-ai-ui-message-stream", "v1")
	s.w.Header().Set("Cache-Control", "no-cache")
	s.w.Header().Set("Connection", "keep-alive")
	s.w.WriteHeader(http.StatusOK)
	s.started = true
	s.send(map[string]string{"type": "start"})
}

func (s *streamWriter) sendRate(tps float64) {
	s.send(map[string]any{
		"type":      "data-inference_rate",
		"data":      map[string]any{"tokensPerSecond": tps},
		"transient": true,
	})
}

func (s *streamWriter) textDelta(id, delta string) {
	now := time.Now()
	if s.genStart.IsZero() {
		s.genStart, s.lastEmit = now, now
	}
	s.tokenCount++
	s.send(map[string]string{"type": "text-delta", "id": id, "delta": delta})
	if elapsed := now.Sub(s.genStart).Seconds(); elapsed > 0 && now.Sub(s.lastEmit) >= 250*time.Millisecond {
		s.sendRate(float64(s.tokenCount) / elapsed)
		s.lastEmit = now
	}
}

func (s *streamWriter) textStart(id string) {
	s.send(map[string]string{"type": "text-start", "id": id})
}

func (s *streamWriter) textEnd(id string) {
	s.send(map[string]string{"type": "text-end", "id": id})
}

func (s *streamWriter) toolInputStart(callID, name string) {
	s.send(map[string]any{
		"type":       "tool-input-start",
		"toolCallId": callID,
		"toolName":   name,
		"dynamic":    true,
	})
}

func (s *streamWriter) toolInputDelta(callID, delta string) {
	s.send(map[string]any{
		"type":           "tool-input-delta",
		"toolCallId":     callID,
		"inputTextDelta": delta,
	})
}

func (s *streamWriter) toolInputAvailable(callID, name string, input json.RawMessage) {
	s.send(map[string]any{
		"type":       "tool-input-available",
		"toolCallId": callID,
		"toolName":   name,
		"input":      input,
		"dynamic":    true,
	})
}

func (s *streamWriter) toolOutputAvailable(callID string, output string) {
	s.send(map[string]any{
		"type":       "tool-output-available",
		"toolCallId": callID,
		"output":     output,
		"dynamic":    true,
	})
}

func (s *streamWriter) errorFrame(msg string) {
	s.send(map[string]string{"type": "error", "errorText": msg})
}

func (s *streamWriter) finish() {
	s.send(map[string]string{"type": "finish"})
	fmt.Fprint(s.w, "data: [DONE]\n\n")
	if s.canFlush {
		s.flusher.Flush()
	}
}

type turn struct {
	text      string
	toolCalls []llm.ToolCall
}

func (s *streamWriter) drain(comp *llm.Completion, textID string) turn {
	var out turn
	var textStarted bool

	for ev := range comp.Events {
		switch ev.Kind {
		case llm.EventText:
			if !textStarted {
				s.textStart(textID)
				textStarted = true
			}
			out.text += ev.Text
			s.textDelta(textID, ev.Text)
		case llm.EventToolStart:
			s.toolInputStart(ev.ToolCallID, ev.ToolName)
		case llm.EventToolArgDelta:
			s.toolInputDelta(ev.ToolCallID, ev.Text)
		case llm.EventToolEnd:
			out.toolCalls = append(out.toolCalls, llm.ToolCall{
				ID:   ev.ToolCallID,
				Name: ev.ToolName,
				Args: ev.Args,
			})
			s.toolInputAvailable(ev.ToolCallID, ev.ToolName, ev.Args)
		}
	}

	if textStarted {
		s.textEnd(textID)
	}
	if usage := comp.Wait(); usage != nil && usage.TokensPerSecond > 0 {
		s.sendRate(usage.TokensPerSecond)
	}

	return out
}
