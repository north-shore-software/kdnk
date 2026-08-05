package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/agent-smith/llm"
	"github.com/north-shore-software/agent-smith/plugin"
)

const (
	startingCredits = 50
	weeklyBurn      = 10
	userTarget      = 500
	minQuality      = -20
	maxQuality      = 20
	statusPlaying   = "playing"
	statusWon       = "won"
	statusLost      = "lost"
)

type RPGPlugin struct{}

func (p *RPGPlugin) Name() string {
	return "rpg"
}

func (p *RPGPlugin) Collections() []*core.Collection {
	col := core.NewBaseCollection("game_state")
	col.Fields.Add(&core.TextField{Name: "owner", Required: true})
	col.Fields.Add(&core.TextField{Name: "conversation", Required: true})
	col.Fields.Add(&core.NumberField{Name: "week"})
	col.Fields.Add(&core.NumberField{Name: "credits"})
	col.Fields.Add(&core.NumberField{Name: "users"})
	col.Fields.Add(&core.NumberField{Name: "quality"})
	col.Fields.Add(&core.TextField{Name: "status"})
	col.AddIndex("idx_game_state_owner_conversation", true, "owner, conversation", "")
	return []*core.Collection{col}
}

func (p *RPGPlugin) SystemPrompt(ctx context.Context, app core.App, s plugin.Session) (string, error) {
	state, err := p.load(app, s)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`You are the game master of a short game about building an AI app. Narrate in second person, two or three sentences per turn, dry and a little cynical.

The run so far:
- week %d
- %dk of inference credits left, burning %dk per week
- %dk of the %dk users needed to win
- model quality %d
- status %s

Use ship_feature when the player ships something to users, and tune_model when they work on the model instead. Every action costs a week. Report what the tool returned as what actually happened — never invent an outcome. The run ends when credits go below zero or the target is reached; if status is not %q, tell the player the run is over.`,
		state.GetInt("week"),
		state.GetInt("credits"),
		weeklyBurn,
		state.GetInt("users"),
		userTarget,
		state.GetInt("quality"),
		state.GetString("status"),
		statusPlaying,
	), nil
}

func (p *RPGPlugin) Tools() []llm.Tool {
	return []llm.Tool{
		{
			Name:        "ship_feature",
			Description: "Ship a feature to users. Costs one week. The outcome depends on model quality and luck.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"feature": {"type": "string", "description": "What is being shipped"},
					"audience": {"type": "number", "description": "How many users it is aimed at, in thousands"}
				},
				"required": ["feature", "audience"]
			}`),
		},
		{
			Name:        "tune_model",
			Description: "Spend a week on evals and prompts instead of shipping. Costs one week and raises model quality.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
	}
}

func (p *RPGPlugin) ExecuteTool(ctx context.Context, app core.App, s plugin.Session, name string, args json.RawMessage) (string, error) {
	state, err := p.load(app, s)
	if err != nil {
		return "", err
	}
	if status := state.GetString("status"); status != statusPlaying {
		return marshal(map[string]any{"refused": "the run is already over", "status": status})
	}

	var outcome map[string]any

	switch name {
	case "ship_feature":
		var input struct {
			Feature  string  `json:"feature"`
			Audience float64 `json:"audience"`
		}
		if err := json.Unmarshal(args, &input); err != nil {
			return "", err
		}
		if input.Feature == "" {
			return "", fmt.Errorf("feature is required")
		}
		if input.Audience <= 0 {
			return "", fmt.Errorf("audience must be greater than zero")
		}
		outcome = ship(state, input.Feature, int(input.Audience))

	case "tune_model":
		outcome = tune(state)

	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}

	if err := app.Save(state); err != nil {
		return "", err
	}

	outcome["week"] = state.GetInt("week")
	outcome["credits"] = state.GetInt("credits")
	outcome["users"] = state.GetInt("users")
	outcome["quality"] = state.GetInt("quality")
	outcome["status"] = state.GetString("status")

	return marshal(outcome)
}

func ship(state *core.Record, feature string, audience int) map[string]any {
	advance(state)

	roll := rand.Intn(100) + state.GetInt("quality")
	outcome := map[string]any{"action": "ship_feature", "feature": feature, "audience": audience, "roll": roll}

	switch {
	case roll >= 85:
		adjustQuality(state, 8)
		credit(state, audience)
		outcome["result"] = "went viral"
		outcome["signups"] = audience
	case roll >= 55:
		partial := audience / 3
		adjustQuality(state, 3)
		credit(state, partial)
		outcome["result"] = "picked up a niche following"
		outcome["signups"] = partial
	default:
		adjustQuality(state, -4)
		outcome["result"] = "hallucinated in front of everyone"
		outcome["signups"] = 0
	}

	settle(state)
	return outcome
}

func tune(state *core.Record) map[string]any {
	advance(state)
	adjustQuality(state, 9)
	settle(state)
	return map[string]any{"action": "tune_model", "result": "the evals actually moved"}
}

func adjustQuality(state *core.Record, delta int) {
	quality := state.GetInt("quality") + delta
	if quality < minQuality {
		quality = minQuality
	}
	if quality > maxQuality {
		quality = maxQuality
	}
	state.Set("quality", quality)
}

func advance(state *core.Record) {
	state.Set("week", state.GetInt("week")+1)
	state.Set("credits", state.GetInt("credits")-weeklyBurn)
}

func credit(state *core.Record, amount int) {
	state.Set("credits", state.GetInt("credits")+amount)
	state.Set("users", state.GetInt("users")+amount)
}

func settle(state *core.Record) {
	switch {
	case state.GetInt("users") >= userTarget:
		state.Set("status", statusWon)
	case state.GetInt("credits") < 0:
		state.Set("status", statusLost)
	}
}

func (p *RPGPlugin) load(app core.App, s plugin.Session) (*core.Record, error) {
	if s.ConversationID == "" {
		return nil, fmt.Errorf("conversation id is required")
	}

	rec, err := app.FindFirstRecordByFilter(
		"game_state",
		"owner = {:o} && conversation = {:c}",
		dbx.Params{"o": s.UserID, "c": s.ConversationID},
	)
	if err == nil {
		return rec, nil
	}

	col, err := app.FindCollectionByNameOrId("game_state")
	if err != nil {
		return nil, err
	}
	rec = core.NewRecord(col)
	rec.Set("owner", s.UserID)
	rec.Set("conversation", s.ConversationID)
	rec.Set("week", 1)
	rec.Set("credits", startingCredits)
	rec.Set("users", 0)
	rec.Set("quality", 0)
	rec.Set("status", statusPlaying)
	if err := app.Save(rec); err != nil {
		if existing, e2 := app.FindFirstRecordByFilter(
			"game_state",
			"owner = {:o} && conversation = {:c}",
			dbx.Params{"o": s.UserID, "c": s.ConversationID},
		); e2 == nil {
			return existing, nil
		}
		return nil, err
	}
	return rec, nil
}

func marshal(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
