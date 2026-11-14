package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/north-shore-software/agent-smith/agentsmith"
	"github.com/north-shore-software/agent-smith/providers"
	"github.com/north-shore-software/agent-smith/providers/gemini"
	"github.com/north-shore-software/agent-smith/providers/ollama"
)

//go:embed web
var webAssets embed.FS

func main() {
	if err := agentsmith.Run(setup); err != nil {
		log.Fatal(err)
	}
}

func setup(r *agentsmith.Registry) error {
	p, err := provider()
	if err != nil {
		return err
	}
	web, err := fs.Sub(webAssets, "web")
	if err != nil {
		return err
	}
	r.Provider(p)
	r.Web(web)
	g := &game{states: agentsmith.Collection[GameState](r, "game_state", agentsmith.Hidden)}
	r.SystemPromptFunc(g.systemPrompt)
	agentsmith.Tool(r, "ship_feature", "Ship a feature to users. Costs one week. The outcome depends on model quality and luck.", g.shipFeature)
	agentsmith.Tool(r, "tune_model", "Spend a week on evals and prompts instead of shipping. Costs one week and raises model quality.", g.tuneModel)
	return nil
}

func provider() (providers.Provider, error) {
	switch set := os.Getenv("AGENT_SMITH_MODEL_SET"); set {
	case "", "local":
		return &ollama.OllamaProvider{Model: "gemma4"}, nil
	case "cloud":
		return &gemini.Provider{Model: "gemini-3.6-flash"}, nil
	default:
		return nil, fmt.Errorf("unknown AGENT_SMITH_MODEL_SET %q (want local or cloud)", set)
	}
}
