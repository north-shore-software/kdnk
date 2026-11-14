---
title: "Example: todo"
---

## What it does

A chat assistant that keeps a task list and helps the user prioritise it by urgency and expected
value, both on a 1–10 scale. The code is in `examples/todo`.

## The app

The whole app is one collection and a system prompt:

```go
type Todo struct {
	Title         string  `json:"title" jsonschema:"What the task is"`
	Urgency       float64 `json:"urgency" jsonschema:"1-10, how soon it must happen"`
	ExpectedValue float64 `json:"expected_value" jsonschema:"1-10, how much it is worth"`
}

func setup(r *kdnk.Registry) error {
	// provider() and web are set up above; see main.go
	r.Provider(p)
	r.SystemPrompt("You help the user prioritise tasks by expected value and urgency, …")
	r.Web(web)
	kdnk.Collection[Todo](r, "todos")
	return nil
}
```

`Collection[Todo]` gives the model `todos_create`, `todos_list`, `todos_get`, `todos_update` and
`todos_delete`. The system prompt names them and tells the model when to use each. No tool is
written by hand.

Todos belong to the user, not to a conversation, so a new conversation sees the same list.

## The web UI

`examples/todo/web/index.html` is a single page with no build step, embedded in the binary and
served with `r.Web`. It:

- signs in as a guest on first load, with `POST /api/guest`, and keeps the guest credentials and
  a conversation ID in `localStorage`
- reloads the conversation from `chat_conversation` and `chat_message` through PocketBase's
  records API
- sends messages to `POST /api/chat`, reads the event stream, and shows each tool call with its
  arguments and result
- has a form to save the guest account with an email and password, through
  `POST /api/guest/upgrade`, and to sign in to an existing account

Clearing `localStorage` abandons the guest account.

## Running it

From the repository root:

```sh
./kdnk.sh setup      # pick Gemini (needs a key) or Ollama, and a port
./kdnk.sh run todo
```

Then open `http://127.0.0.1:8090/` (or the port you chose) and try
`add: file taxes, urgency 8, value 9`. `./kdnk.sh clear todo` deletes the example's data.
