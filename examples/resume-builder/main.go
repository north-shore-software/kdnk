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

type Job struct {
	Company  string `json:"company" jsonschema:"The employer's name"`
	Title    string `json:"title" jsonschema:"The job title"`
	Location string `json:"location" jsonschema:"Where the job was, e.g. a city or Remote"`
	Start    string `json:"start" jsonschema:"When the job started, as YYYY-MM"`
	End      string `json:"end" jsonschema:"When the job ended, as YYYY-MM, or empty if it is the current job"`
	Summary  string `json:"summary" jsonschema:"What the user did and achieved there, as short bullet points"`
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
		"You help the user build the work-experience section of their resume.",
		"Use jobs_create to record a job, jobs_list and jobs_get to read what is already recorded,",
		"jobs_update to change one, and jobs_delete to remove one.",
		"When the user describes a job, ask for anything missing (company, title, location, start and end month) before recording it,",
		"and turn what they did into short, concrete bullet points that start with a verb.",
		"After a tool returns, tell the user what happened in plain language.",
	}, " "))
	r.Web(web)
	kdnk.Collection[Job](r, "jobs")
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
