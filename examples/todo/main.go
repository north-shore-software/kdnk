package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/north-shore-software/kdnk/kdnk"
	"github.com/north-shore-software/kdnk/providers"
)

//go:embed web
var webAssets embed.FS

type Todo struct {
	Title         string  `json:"title" jsonschema:"What the task is"`
	Urgency       float64 `json:"urgency" jsonschema:"1-10, how soon it must happen"`
	ExpectedValue float64 `json:"expected_value" jsonschema:"1-10, how much it is worth"`
}

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
	web, err := fs.Sub(webAssets, "web")
	if err != nil {
		return err
	}
	r.Provider(p)
	r.SystemPrompt(strings.Join([]string{
		"You help the user prioritise tasks by expected value and urgency, both on a 1-10 scale.",
		"Use todos_create to record a task, todos_list and todos_get to read what is already recorded,",
		"todos_update to change a task, and todos_delete to remove one.",
		"After a tool returns, tell the user what happened in plain language.",
	}, " "))
	r.Web(web)
	kdnk.Collection[Todo](r, "todos")
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
