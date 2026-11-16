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

//go:embed all:web/dist
var webAssets embed.FS

type Profile struct {
	Name     string `json:"name" jsonschema:"The user's full name"`
	Headline string `json:"headline" jsonschema:"One line describing who they are professionally"`
	Email    string `json:"email" jsonschema:"Contact email"`
	Phone    string `json:"phone" jsonschema:"Contact phone number"`
	Location string `json:"location" jsonschema:"City and region, or Remote"`
	Links    string `json:"links" jsonschema:"Portfolio, LinkedIn, GitHub and similar URLs, one per line"`
}

type PastRole struct {
	Company  string `json:"company" jsonschema:"The employer's name"`
	Title    string `json:"title" jsonschema:"The job title"`
	Location string `json:"location" jsonschema:"Where the role was, e.g. a city or Remote"`
	Start    string `json:"start" jsonschema:"When the role started, as YYYY-MM"`
	End      string `json:"end" jsonschema:"When the role ended, as YYYY-MM, or empty if it is current"`
	Summary  string `json:"summary" jsonschema:"What the user did there, as short bullet points"`
}

type WarStory struct {
	RoleID    string `json:"role_id" jsonschema:"ID of the past_roles record this happened in, or empty"`
	Title     string `json:"title" jsonschema:"A short label for the story"`
	Situation string `json:"situation" jsonschema:"The context and the problem"`
	Action    string `json:"action" jsonschema:"What the user personally did"`
	Result    string `json:"result" jsonschema:"What changed because of it, with numbers where possible"`
}

type Skill struct {
	Name     string `json:"name" jsonschema:"The skill"`
	Category string `json:"category" jsonschema:"e.g. language, tool, domain, leadership"`
	Evidence string `json:"evidence" jsonschema:"Where the user showed it, naming roles or war stories"`
}

type JobDescription struct {
	Company string `json:"company" jsonschema:"The hiring company"`
	Title   string `json:"title" jsonschema:"The job title"`
	Text    string `json:"text" jsonschema:"The full job description as the user gave it"`
}

type Resume struct {
	JobDescriptionID string `json:"job_description_id" jsonschema:"ID of the job_descriptions record it targets"`
	Content          string `json:"content" jsonschema:"The resume, as markdown"`
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
	web, err := fs.Sub(webAssets, "web/dist/app")
	if err != nil {
		return err
	}
	r.Provider(p)
	r.SystemPrompt(strings.Join([]string{
		"You help the user build a complete audit of their working life, then write resumes from it.",
		"Record past roles in past_roles, specific accomplishments in war_stories (situation, action, result),",
		"skills in skills with evidence, and contact details in profile.",
		"Ask follow-up questions to draw out concrete results and numbers; never invent facts.",
		"When the user gives a job description, save it in job_descriptions.",
		"To write a resume, read profile and list every past role, war story and skill,",
		"then write a markdown resume tailored to that job description using only recorded facts,",
		"and save it as a new record in resumes.",
		"After a tool returns, tell the user what happened in plain language.",
	}, " "))
	r.Web(web)
	kdnk.Singleton[Profile](r, "profile")
	kdnk.Collection[PastRole](r, "past_roles")
	kdnk.Collection[WarStory](r, "war_stories")
	kdnk.Collection[Skill](r, "skills")
	kdnk.Collection[JobDescription](r, "job_descriptions")
	kdnk.Collection[Resume](r, "resumes", kdnk.AppendOnly)
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
