---
title: "Example: rpg"
---

## What it does

A short game about building an AI app, with the model as game master. Each turn the player ships
a feature or tunes the model. Each action costs a week of inference credits, and the run is won at
500k users or lost when the credits run out. The code is in `examples/rpg`.

## The app

The game state is a collection that only Go code can see:

```go
type GameState struct {
	Week    int    `json:"week"`
	Credits int    `json:"credits"`
	Users   int    `json:"users"`
	Quality int    `json:"quality"`
	Status  string `json:"status"`
}

g := &game{states: agentsmith.Collection[GameState](r, "game_state", agentsmith.Hidden)}
r.SystemPromptFunc(g.systemPrompt)
agentsmith.Tool(r, "ship_feature", "Ship a feature to users. …", g.shipFeature)
agentsmith.Tool(r, "tune_model", "Spend a week on evals and prompts instead of shipping. …", g.tuneModel)
```

- **`Hidden`** means the model gets no tools for `game_state`. It can't edit the score.
- **`SystemPromptFunc`** loads the user's game, creating it on the first request, and writes the
  current week, credits, users, quality and status into the prompt on every request.
- **`ship_feature`** takes a `feature` and an `audience` in thousands. It rolls a random number
  plus the model quality, and updates the state with the outcome.
- **`tune_model`** takes no arguments (`struct{}`) and raises the quality.

The rules live in the tool handlers, and the tools return the outcome as JSON. The prompt tells
the model to narrate what the tool returned and never invent an outcome.

Limits of the current design:

- The game is per user, not per conversation, because collections are scoped by user only. A
  new conversation continues the same game.
- There is no way to start over once a run is won or lost.

## The web UI

`examples/rpg/web/index.html` is the same page as the [todo example](/examples/todo/#the-web-ui)
with a terminal-style theme.

## Running it

From the repository root:

```sh
./smith.sh setup
./smith.sh run rpg
```

Open the page and send `let's start`. `./smith.sh clear rpg` deletes the game.
