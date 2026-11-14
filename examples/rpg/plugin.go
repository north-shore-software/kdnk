package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/agent-smith/agentsmith"
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

type GameState struct {
	Week    int    `json:"week"`
	Credits int    `json:"credits"`
	Users   int    `json:"users"`
	Quality int    `json:"quality"`
	Status  string `json:"status"`
}

type ShipFeatureInput struct {
	Feature  string  `json:"feature" jsonschema:"What is being shipped"`
	Audience float64 `json:"audience" jsonschema:"How many users it is aimed at, in thousands"`
}

type game struct {
	states *agentsmith.Store[GameState]
}

func (g *game) systemPrompt(ctx context.Context, app core.App, s agentsmith.Session) (string, error) {
	rec, err := g.load(ctx, s)
	if err != nil {
		return "", err
	}
	state := rec.Data

	return fmt.Sprintf(`You are the game master of a short game about building an AI app. Narrate in second person, two or three sentences per turn, dry and a little cynical.

The run so far:
- week %d
- %dk of inference credits left, burning %dk per week
- %dk of the %dk users needed to win
- model quality %d
- status %s

Use ship_feature when the player ships something to users, and tune_model when they work on the model instead. Every action costs a week. Report what the tool returned as what actually happened — never invent an outcome. The run ends when credits go below zero or the target is reached; if status is not %q, tell the player the run is over.`,
		state.Week,
		state.Credits,
		weeklyBurn,
		state.Users,
		userTarget,
		state.Quality,
		state.Status,
		statusPlaying,
	), nil
}

func (g *game) shipFeature(ctx context.Context, app core.App, s agentsmith.Session, in ShipFeatureInput) (any, error) {
	if in.Feature == "" {
		return nil, fmt.Errorf("feature is required")
	}
	if in.Audience <= 0 {
		return nil, fmt.Errorf("audience must be greater than zero")
	}
	return g.act(ctx, s, func(state *GameState) map[string]any {
		return ship(state, in.Feature, int(in.Audience))
	})
}

func (g *game) tuneModel(ctx context.Context, app core.App, s agentsmith.Session, _ struct{}) (any, error) {
	return g.act(ctx, s, tune)
}

func (g *game) act(ctx context.Context, s agentsmith.Session, action func(*GameState) map[string]any) (any, error) {
	rec, err := g.load(ctx, s)
	if err != nil {
		return nil, err
	}
	state := rec.Data
	if state.Status != statusPlaying {
		return map[string]any{"refused": "the run is already over", "status": state.Status}, nil
	}

	outcome := action(&state)

	if _, err := g.states.Update(ctx, s, rec.ID, state); err != nil {
		return nil, err
	}

	outcome["week"] = state.Week
	outcome["credits"] = state.Credits
	outcome["users"] = state.Users
	outcome["quality"] = state.Quality
	outcome["status"] = state.Status

	return outcome, nil
}

func ship(state *GameState, feature string, audience int) map[string]any {
	advance(state)

	roll := rand.Intn(100) + state.Quality
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

func tune(state *GameState) map[string]any {
	advance(state)
	adjustQuality(state, 9)
	settle(state)
	return map[string]any{"action": "tune_model", "result": "the evals actually moved"}
}

func adjustQuality(state *GameState, delta int) {
	state.Quality = min(max(state.Quality+delta, minQuality), maxQuality)
}

func advance(state *GameState) {
	state.Week++
	state.Credits -= weeklyBurn
}

func credit(state *GameState, amount int) {
	state.Credits += amount
	state.Users += amount
}

func settle(state *GameState) {
	switch {
	case state.Users >= userTarget:
		state.Status = statusWon
	case state.Credits < 0:
		state.Status = statusLost
	}
}

func (g *game) load(ctx context.Context, s agentsmith.Session) (agentsmith.Record[GameState], error) {
	recs, err := g.states.List(ctx, s, 1)
	if err != nil {
		return agentsmith.Record[GameState]{}, err
	}
	if len(recs) > 0 {
		return recs[0], nil
	}
	return g.states.Create(ctx, s, GameState{
		Week:    1,
		Credits: startingCredits,
		Quality: 0,
		Status:  statusPlaying,
	})
}
