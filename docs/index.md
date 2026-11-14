---
title: kdnk
---

## What it is

kdnk is a Go library for apps whose main interface is a chat with an LLM. You write a setup
function that registers a model provider, a system prompt, collections of records and tools.
kdnk turns that into a single server binary that handles:

- user accounts, including guest users who can later add an email and password
- conversation history, stored per user
- a streaming chat endpoint that speaks the
  [AI SDK UI message stream](https://ai-sdk.dev/docs/ai-sdk-ui/stream-protocol) protocol
- tool calling, with tools generated for each collection and tools you write yourself
- an optional embedded web UI

It is built on [PocketBase](https://pocketbase.io/), so the binary also has PocketBase's SQLite
database, admin dashboard and command line.

## How it fits together

- **Your app** calls `kdnk.Run(setup)` from `main`. `setup` gets an `*kdnk.Registry`
  and registers what the app needs. See [Building an app](/apps/).
- **The registry** is checked when `setup` returns. Mistakes such as two tools with the same name
  make `Run` return an error before anything starts.
- **The server** is a PocketBase app. When it starts serving, it creates or extends your
  collections' tables, and adds the guest and chat routes. See
  [Server configuration](/server/).
- **The provider** sends each chat request to a model and streams the answer back. Gemini and
  Ollama are included, and you can write your own. See [LLM providers](/providers/).
- **A client**, such as the web UI in each example, signs in as a guest and posts to
  `/api/chat`.

## Where to start

- [Quickstart](/quickstart/) walks through a small app.
- The [examples](/examples/todo/) are complete apps with a web UI. Run them from the repository
  root with `./kdnk.sh setup`, then `./kdnk.sh run`.
