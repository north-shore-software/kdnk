package main

import (
	"embed"
	"io/fs"
	"log"
	"os"

	"github.com/north-shore-software/agent-smith/gemini"
	"github.com/north-shore-software/agent-smith/llm"
	"github.com/north-shore-software/agent-smith/ollama"
	"github.com/north-shore-software/agent-smith/server"
)

//go:embed web
var webAssets embed.FS

func main() {
	if raw := os.Getenv("AGENT_SMITH_MODEL_SET"); raw != "" {
		set, err := llm.ParseModelSet(raw)
		if err != nil {
			log.Fatal(err)
		}
		llm.SetActiveModelSet(set)
	}

	llm.SetProviderFactory(func(model string) llm.Provider {
		provider, err := llm.ProviderFor(model)
		if err != nil {
			return llm.ErrorProvider(err)
		}
		switch provider {
		case llm.ProviderGemini:
			return &gemini.Provider{Model: model}
		default:
			return &ollama.OllamaProvider{Model: model}
		}
	})

	web, err := fs.Sub(webAssets, "web")
	if err != nil {
		log.Fatal(err)
	}

	app := server.New(server.Config{
		Plugin: &RPGPlugin{},
		Web:    web,
	})

	server.EnsureReady()

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
