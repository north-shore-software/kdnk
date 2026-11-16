---
title: Quickstart
---

This walkthrough builds a small kdnk app: a chat assistant that can save notes, list them,
and tell the time.

## Requirements

- Go 1.27 or newer
- A Gemini API key, from [Google AI Studio](https://aistudio.google.com/apikey)

## Create a module

```sh
mkdir notes && cd notes
go mod init notes
go get github.com/north-shore-software/kdnk
```

## Write the app

A kdnk app is a **setup function**. At startup, kdnk calls it once with a registry,
and the function registers what the app needs: a model provider, a system prompt, collections and
tools. Registration closes when the function returns. Create `main.go`:

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"

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
	r.SystemPrompt("You are a note-taking assistant. Use get_time when the user asks about the time.")
	kdnk.Collection[Note](r, "notes")
	kdnk.Tool(r, "get_time", "Get the current date and time on the server", getTime)
	return nil
}

func getTime(ctx context.Context, app core.App, s kdnk.Session, _ struct{}) (any, error) {
	return time.Now().Format(time.RFC1123), nil
}
```

- `Run` calls `setup`, then starts the command line (`serve` runs the server).
- `Provider` sets the model that answers chat requests. `providers.Gemini` reads `GEMINI_API_KEY`.
- `SystemPrompt` is sent to the model at the start of every chat request.
- `Collection` declares a collection of `Note` records named `notes`. kdnk creates the
  table, and gives the model tools to manage it: `notes_create`, `notes_list`, `notes_get`,
  `notes_update` and `notes_delete`. Their arguments come from the fields of `Note`.
  `notes_list` returns up to a limit the model chooses. Each record belongs to the user who
  created it, and a user only ever sees their own. The tools return a `version` with each record;
  `notes_update` and `notes_delete` take the version the model last read, and if the record has
  changed since they change nothing and return the current record.
- `Tool` registers a tool you write yourself, for anything that isn't a collection operation. The
  model is shown its name, its description, and a JSON Schema generated from the handler's
  argument type; `get_time` takes none (`struct{}`). When the model calls it, kdnk decodes
  the arguments into that type, runs the handler, and passes the result back to the model as JSON.

A collection can be restricted to append-only, where the model can add records but not change or
delete them, to read-only, where only your code writes to it, or hidden, where the model gets no
tools for it at all.

## Start the server

```sh
go mod tidy
GEMINI_API_KEY=your-key go run . serve
```

The server listens on `http://127.0.0.1:8090` and exposes `POST /api/guest` to create a guest user
and `POST /api/chat` to stream replies in the
[AI SDK UI message stream](https://ai-sdk.dev/docs/ai-sdk-ui/stream-protocol) format. A web
frontend built with the AI SDK's `useChat` can talk to the server directly.

:::note
An OpenAI-compatible endpoint, so that tools such as [`llm`](https://llm.datasette.io/) and
`aichat` can talk to a kdnk server, is planned but not built yet.
:::

## Next steps

- [Concepts](/concepts/setup/) cover the setup function, sessions, collections and tools in depth.
- [LLM providers](/concepts/providers/) covers Gemini and Ollama.
- [Example: todo](/examples/todo/) is a complete app with a web UI.
