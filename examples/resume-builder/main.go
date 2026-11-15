package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/north-shore-software/kdnk/kdnk"
	"github.com/north-shore-software/kdnk/providers"
)

//go:embed all:web/dist
var webAssets embed.FS

func main() {
	if err := kdnk.Run(setup); err != nil {
		log.Fatal(err)
	}
}

func setup(r *kdnk.Registry) error {
	p, err := provider()
	if err != nil {
		return err
	}
	web, err := fs.Sub(webAssets, "web/dist/app")
	if err != nil {
		return err
	}
	r.Provider(p)
	r.Web(web)
	g := &game{states: kdnk.Collection[GameState](r, "game_state", kdnk.Hidden)}
	r.SystemPromptFunc(g.systemPrompt)
	kdnk.Tool(r, "ship_feature", "Ship a feature to users. Costs one week. The outcome depends on model quality and luck.", g.shipFeature)
	kdnk.Tool(r, "tune_model", "Spend a week on evals and prompts instead of shipping. Costs one week and raises model quality.", g.tuneModel)
	return nil
}

func provider() (providers.Provider, error) {
	switch set := os.Getenv("KDNK_MODEL_SET"); set {
	case "", "local":
		return &providers.Ollama{Model: "gemma4"}, nil
	case "cloud":
		return &providers.Gemini{Model: "gemini-3.6-flash"}, nil
	default:
		return nil, fmt.Errorf("unknown KDNK_MODEL_SET %q (want local or cloud)", set)
	}
}
