package server

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/kdnk/internal/chat"
	"github.com/north-shore-software/kdnk/internal/plugin"
	"github.com/north-shore-software/kdnk/providers"
)

const maxToolSteps = 4

type chatRequest struct {
	ID       string         `json:"id"`
	Messages []chat.Message `json:"messages"`
}

func handleChat(app core.App, p plugin.Plugin, provider providers.Provider) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req := chatRequest{}
		if err := e.BindBody(&req); err != nil {
			return e.BadRequestError("invalid chat request body", err)
		}
		if len(req.Messages) == 0 {
			return e.BadRequestError("messages required", nil)
		}
		if req.ID == "" {
			return e.BadRequestError("conversation id required", nil)
		}
		if e.Auth == nil {
			return e.UnauthorizedError("authentication required", nil)
		}

		ctx := e.Request.Context()
		sess := plugin.Session{ConversationID: req.ID, UserID: e.Auth.Id}

		model := provider.ModelName()

		conv, err := chat.FindOrCreateConversation(ctx, app, sess.UserID, sess.ConversationID)
		if err != nil {
			return e.InternalServerError("conversation lookup failed", err)
		}

		dbMsgs, err := chat.LoadMessages(ctx, app, conv)
		if err != nil {
			return e.InternalServerError("history load failed", err)
		}

		seq := len(dbMsgs)
		newMsgs := chat.ExtractNewMessages(dbMsgs, req.Messages)
		for _, m := range newMsgs {
			if err := chat.PersistMessage(ctx, app, conv, m, seq, ""); err != nil {
				return e.InternalServerError("message persist failed", err)
			}
			seq++
		}

		var msgs []providers.Message
		sysPrompt, err := p.SystemPrompt(ctx, app, sess)
		if err != nil {
			return e.InternalServerError("system prompt failed", err)
		}
		if sysPrompt != "" {
			msgs = append(msgs, providers.Message{Role: "system", Content: sysPrompt})
		}
		msgs = append(msgs, chat.Hydrate(append(dbMsgs, newMsgs...))...)
		if len(msgs) == 0 {
			return e.BadRequestError("messages required", nil)
		}

		tools := p.Tools()

		comp, err := provider.Stream(ctx, msgs, tools)
		if err != nil {
			return e.InternalServerError("llm stream failed", err)
		}

		sw := newStreamWriter(e.Response)
		sw.start()

		persist := func(m chat.Message) {
			if len(m.Parts) == 0 {
				return
			}
			if err := chat.PersistMessage(ctx, app, conv, m, seq, model); err != nil {
				app.Logger().Error("assistant message persist failed", "conversation", conv.Id, "error", err)
			}
			seq++
		}

		for step := 0; ; step++ {
			textID := fmt.Sprintf("txt-%s-%d", conv.Id, seq)
			t := sw.drain(comp, textID)

			stored := chat.Message{ID: textID, Role: "assistant"}
			if t.text != "" {
				stored.Parts = append(stored.Parts, chat.TextPart(t.text))
			}

			if len(t.toolCalls) == 0 || step+1 >= maxToolSteps {
				persist(stored)
				break
			}

			msgs = append(msgs, providers.Message{Role: "assistant", Content: t.text, ToolCalls: t.toolCalls})

			for _, tc := range t.toolCalls {
				out, err := p.ExecuteTool(ctx, app, sess, tc.Name, tc.Args)
				if err != nil {
					out = fmt.Sprintf(`{"error":%q}`, err.Error())
				}
				sw.toolOutputAvailable(tc.ID, out)

				if part, err := chat.ToolPart(tc, out); err == nil {
					stored.Parts = append(stored.Parts, part)
				}
				msgs = append(msgs, providers.Message{
					Role:       "tool",
					Content:    out,
					ToolCallID: tc.ID,
					ToolName:   tc.Name,
				})
			}

			persist(stored)

			next := tools
			if step+2 >= maxToolSteps {
				next = nil
			}
			comp, err = provider.Stream(ctx, msgs, next)
			if err != nil {
				sw.errorFrame(err.Error())
				break
			}
		}

		sw.finish()
		return nil
	}
}
