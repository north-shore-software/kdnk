---
title: Quickstart
---

This walkthrough builds a small agent-smith app: a chat assistant that can save notes, list them,
and tell the time. You will run it locally and talk to it from the terminal.

## Requirements

- Go 1.24.1 or newer
- A Gemini API key, from [Google AI Studio](https://aistudio.google.com/apikey)

## Create a module

```sh
mkdir notes && cd notes
go mod init notes
go get github.com/north-shore-software/agent-smith
```

## Write the app

An agent-smith app is a **setup function**. At startup, agent-smith calls it once with a registry,
and the function registers what the app needs: a model provider, a system prompt, collections and
tools. Registration closes when the function returns. Create `main.go`:

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/agent-smith/agentsmith"
	"github.com/north-shore-software/agent-smith/providers/gemini"
)

type Note struct {
	Text string `json:"text" jsonschema:"the note"`
}

func main() {
	if err := agentsmith.Run(setup); err != nil {
		log.Fatal(err)
	}
}

func setup(r *agentsmith.Registry) error {
	r.Provider(&gemini.Provider{Model: "gemini-3.6-flash"})
	r.SystemPrompt("You are a note-taking assistant. Use get_time when the user asks about the time.")
	agentsmith.Collection[Note](r, "notes")
	agentsmith.Tool(r, "get_time", "Get the current date and time on the server", getTime)
	return nil
}

func getTime(ctx context.Context, app core.App, s agentsmith.Session, _ struct{}) (any, error) {
	return time.Now().Format(time.RFC1123), nil
}
```

- `Run` calls `setup`, then starts the command line: `serve` runs the server and `chat` talks
  to it.
- `Provider` sets the model that answers chat requests. `gemini.Provider` reads `GEMINI_API_KEY`.
- `SystemPrompt` is sent to the model at the start of every chat request.
- `Collection` declares a collection of `Note` records named `notes`. agent-smith creates the
  table, and gives the model tools to manage it: `notes_create`, `notes_list`, `notes_get`,
  `notes_update` and `notes_delete`. Their arguments come from the fields of `Note`.
  `notes_list` returns up to a limit the model chooses. Each record belongs to the user who
  created it, and a user only ever sees their own.
- `Tool` registers a tool you write yourself, for anything that isn't a collection operation. The
  model is shown its name, its description, and a JSON Schema generated from the handler's
  argument type; `get_time` takes none (`struct{}`). When the model calls it, agent-smith decodes
  the arguments into that type, runs the handler, and passes the result back to the model as JSON.

A collection can be restricted to append-only, where the model can add records but not change or
delete them, to read-only, where only your code writes to it, or hidden, where the model gets no
tools for it at all.

## Start the server

```sh
go mod tidy
GEMINI_API_KEY=your-key go run . serve
```

The server listens on `http://127.0.0.1:8090`. On first start it prints a link for creating an
admin account for the database dashboard; you do not need it for this walkthrough.

## Chat with it

In a second terminal, in the same directory:

```sh
go run . chat
```

`chat` connects to the server at `http://127.0.0.1:8090` (change it with `--url`), signs in as a
new guest user, and starts a conversation. Tool calls are shown in brackets. Press Ctrl-D to quit.

```text
> remember that the bins go out on Thursday
[notes_create]
Noted: the bins go out on Thursday.
> what time is it?
[get_time]
It's 14:02 on Friday 13 November.
> what are my notes?
[notes_list]
You have one note: the bins go out on Thursday.
```

`chat` talks to the server over the same HTTP API as any other client: `POST /api/guest` to get a
guest user, then `POST /api/chat`, which streams replies in the
[AI SDK UI message stream](https://ai-sdk.dev/docs/ai-sdk-ui/stream-protocol) format. A web
frontend built with the AI SDK's `useChat` can talk to the same server.

:::note
An OpenAI-compatible endpoint, so that tools such as [`llm`](https://llm.datasette.io/) and
`aichat` can talk to an agent-smith server, is planned but not built yet.
:::

## Next steps

- [Building an app](/apps/) covers collections, sessions and tools in depth.
- [LLM providers](/providers/) covers Ollama and custom providers.
- [Example: todo](/examples/todo/) is a complete app with a web UI.
