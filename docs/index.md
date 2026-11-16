---
title: kdnk
---

kdnk is a Go library for apps whose main interface is a chat with an LLM. You register a model
provider, a system prompt, collections of records and tools. kdnk turns that into a single server
binary with user accounts, conversation history, a streaming chat endpoint and tool calling, built
on [PocketBase](https://pocketbase.io/).

```go
package main

import (
	"log"

	"github.com/north-shore-software/kdnk/kdnk"
	"github.com/north-shore-software/kdnk/providers"
)

type Note struct {
	Text string `json:"text" jsonschema:"the note"`
}

func main() {
	if err := kdnk.Run(setup); err != nil {
		log.Fatal(err)
	}
}

func setup(r *kdnk.Registry) error {
	r.Provider(&providers.Gemini{Model: "gemini-3.6-flash"})
	r.SystemPrompt("You are a note-taking assistant.")
	kdnk.Collection[Note](r, "notes")
	return nil
}
```

The model can now create, list, update and delete the signed-in user's notes.

- [Quickstart](/quickstart/) builds and runs this app.
- [Concepts](/concepts/setup/) explain each part.
- [Examples](/examples/todo/) are complete apps with a web UI.
