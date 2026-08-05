package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/agent-smith/llm"
	"github.com/north-shore-software/agent-smith/plugin"
)

type TodoPlugin struct{}

func (p *TodoPlugin) Name() string {
	return "todo"
}

func (p *TodoPlugin) Collections() []*core.Collection {
	col := core.NewBaseCollection("todos")
	col.Fields.Add(&core.TextField{Name: "owner", Required: true})
	col.Fields.Add(&core.TextField{Name: "title", Required: true})
	col.Fields.Add(&core.NumberField{Name: "urgency"})
	col.Fields.Add(&core.NumberField{Name: "expected_value"})
	col.Fields.Add(&core.TextField{Name: "status"})
	col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	col.AddIndex("idx_todos_owner", false, "owner", "")
	return []*core.Collection{col}
}

func (p *TodoPlugin) SystemPrompt(ctx context.Context, app core.App, s plugin.Session) (string, error) {
	return strings.Join([]string{
		"You help the user prioritise tasks by expected value and urgency, both on a 1-10 scale.",
		"Use add_todo to record a task and list_todos to read what is already recorded.",
		"After a tool returns, tell the user what happened in plain language.",
	}, " "), nil
}

func (p *TodoPlugin) Tools() []llm.Tool {
	return []llm.Tool{
		{
			Name:        "add_todo",
			Description: "Record a new task with its urgency and expected value",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"title": {"type": "string", "description": "What the task is"},
					"urgency": {"type": "number", "description": "1-10, how soon it must happen"},
					"expected_value": {"type": "number", "description": "1-10, how much it is worth"}
				},
				"required": ["title"]
			}`),
		},
		{
			Name:        "list_todos",
			Description: "List all recorded tasks with their urgency and expected value",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
	}
}

func (p *TodoPlugin) ExecuteTool(ctx context.Context, app core.App, s plugin.Session, name string, args json.RawMessage) (string, error) {
	switch name {
	case "add_todo":
		return p.addTodo(app, s.UserID, args)
	case "list_todos":
		return p.listTodos(app, s.UserID)
	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}
}

func (p *TodoPlugin) addTodo(app core.App, ownerID string, args json.RawMessage) (string, error) {
	var input struct {
		Title         string  `json:"title"`
		Urgency       float64 `json:"urgency"`
		ExpectedValue float64 `json:"expected_value"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	if input.Title == "" {
		return "", fmt.Errorf("title is required")
	}

	col, err := app.FindCollectionByNameOrId("todos")
	if err != nil {
		return "", err
	}
	rec := core.NewRecord(col)
	rec.Set("owner", ownerID)
	rec.Set("title", input.Title)
	rec.Set("urgency", input.Urgency)
	rec.Set("expected_value", input.ExpectedValue)
	rec.Set("status", "pending")
	if err := app.Save(rec); err != nil {
		return "", err
	}

	return marshal(map[string]any{"status": "created", "id": rec.Id, "title": input.Title})
}

func (p *TodoPlugin) listTodos(app core.App, ownerID string) (string, error) {
	recs, err := app.FindRecordsByFilter(
		"todos",
		"owner = {:owner}",
		"created", 0, 0,
		dbx.Params{"owner": ownerID},
	)
	if err != nil {
		return "", err
	}

	items := make([]map[string]any, 0, len(recs))
	for _, r := range recs {
		items = append(items, map[string]any{
			"id":             r.Id,
			"title":          r.GetString("title"),
			"urgency":        r.GetFloat("urgency"),
			"expected_value": r.GetFloat("expected_value"),
			"status":         r.GetString("status"),
		})
	}

	return marshal(map[string]any{"todos": items})
}

func marshal(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
