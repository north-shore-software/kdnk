---
title: Building an app
---

## The setup function

An app is a function that takes an `*agentsmith.Registry`. Pass it to `agentsmith.Run` from
`main`:

```go
func main() {
	if err := agentsmith.Run(setup); err != nil {
		log.Fatal(err)
	}
}

func setup(r *agentsmith.Registry) error {
	r.Provider(&gemini.Provider{Model: "gemini-3.6-flash"})
	r.SystemPrompt("You are a note-taking assistant.")
	agentsmith.Collection[Note](r, "notes")
	return nil
}
```

`Run` calls `setup` once, checks what it registered, and then runs the PocketBase command line
(`serve`, `migrate`, `superuser` and so on). It returns an error, without starting anything, if:

- `setup` returns an error
- `Provider` was not called, was called with nil, or was called more than once
- `Web` was called more than once
- `SystemPrompt` and `SystemPromptFunc` were called more than once between them
- a collection's or a tool's type is invalid
- two collections, or two tools, have the same name
- a collection is named `users`, `chat_conversation` or `chat_message`, or its name is empty or
  starts with an underscore

Every problem found is reported in the one error.

## Sessions

Tool handlers and the system prompt function receive an `agentsmith.Session`:

- `UserID` is the ID of the signed-in user's record in the `users` collection.
- `ConversationID` is the conversation ID the client sent with the chat request.

Collections are scoped by `UserID`. Nothing is scoped by `ConversationID` yet, so state you store
in a collection is shared by all of a user's conversations.

## Collections

`agentsmith.Collection[T](r, name, opts...)` declares a collection whose fields come from the
struct `T`, and returns a `*agentsmith.Store[T]` for using it from Go.

```go
type Todo struct {
	Title   string  `json:"title" jsonschema:"What the task is"`
	Urgency float64 `json:"urgency" jsonschema:"1-10, how soon it must happen"`
}

todos := agentsmith.Collection[Todo](r, "todos")
```

**Fields.** Each exported field becomes a column, named by its `json` tag, or by the Go field name
if the tag has no name. Fields tagged `json:"-"` are skipped. The `jsonschema` tag is the field's
description in the generated tools. These types are supported, including named types with these
underlying kinds:

| Go type | PocketBase field |
|---|---|
| `string` | text |
| `bool` | bool |
| signed and unsigned integers | number, integers only |
| `float32`, `float64` | number |
| `time.Time` | date |

Embedded fields, any other type, the names `id`, `owner` and `created`, and two fields with the
same name make `Run` return an error.

**Hidden fields.** Every record also has an `owner`, the user who created it, and a `created`
time. Records are deleted when their owner is.

**Tables.** When the server starts serving, the table is created if it doesn't exist. If it does,
missing columns are added. A column whose type differs from the struct makes serving fail.
Columns are never changed or removed.

**Generated tools.** By default the model gets five tools for a collection called `todos`:

| Tool | Arguments | Result |
|---|---|---|
| `todos_create` | the struct's fields | the new record |
| `todos_list` | `limit`, optional, 1 to 100, default 20 | records, newest first |
| `todos_get` | `id` | the record |
| `todos_update` | `id` and all the struct's fields | the updated record |
| `todos_delete` | `id` | `{"deleted": "<id>"}` |

A record is returned to the model as the struct's JSON plus `id` and `created`. `todos_update`
replaces every field. Every tool only sees the session user's records; another user's ID is
reported as not found.

**Restrictions.** Pass an option to limit which tools are generated. The last one passed wins.
Restrictions apply to the model only; Go code can always use every method of the store.

| Option | Tools |
|---|---|
| none | create, list, get, update, delete |
| `agentsmith.AppendOnly` | create, list, get |
| `agentsmith.ReadOnly` | list, get |
| `agentsmith.Hidden` | none |

**Using a collection from Go.** The `Store[T]` methods are scoped to a session's user, and work
once `Run` has started the app. Calling them earlier returns an error.

- `Create(ctx, s, v)` adds a record and returns it.
- `List(ctx, s, limit)` returns up to `limit` records, newest first. `limit` must be positive.
- `Get(ctx, s, id)` returns a record.
- `Update(ctx, s, id, v)` replaces a record's fields and returns it.
- `Delete(ctx, s, id)` removes a record.

`Get`, `Update` and `Delete` return `agentsmith.ErrNotFound` if the record doesn't exist or
belongs to another user. Each `Record[T]` has `ID`, `Created` and `Data`.

## System prompt

`r.SystemPrompt(text)` sets a prompt sent at the start of every chat request.

`r.SystemPromptFunc(fn)` builds it instead, once per chat request, before the first model call.
Use it when the prompt depends on stored state:

```go
r.SystemPromptFunc(func(ctx context.Context, app core.App, s agentsmith.Session) (string, error) {
	rec, err := states.Get(ctx, s, id)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("The player is in week %d.", rec.Data.Week), nil
})
```

An empty prompt sends no system message. An error from the function fails the request with HTTP
500. Set one or the other, once.

## Tools

`agentsmith.Tool(r, name, description, handler)` registers a tool you write yourself:

```go
type ShipInput struct {
	Feature  string  `json:"feature" jsonschema:"What is being shipped"`
	Audience float64 `json:"audience" jsonschema:"How many users it is aimed at, in thousands"`
}

agentsmith.Tool(r, "ship_feature", "Ship a feature to users.",
	func(ctx context.Context, app core.App, s agentsmith.Session, in ShipInput) (any, error) {
		return map[string]any{"result": "shipped " + in.Feature}, nil
	})
```

The tool's JSON Schema is generated from the handler's argument type, which must be a struct.
Field names, `jsonschema` descriptions and supported types follow the same rules as collections.
Every field is a required argument unless its `json` tag has `omitempty`. A tool with no
arguments takes `struct{}`.

The handler gets the PocketBase app, so it can also read and write the database directly.

## Running tools

When the model calls a tool, the server:

1. decodes the arguments into the handler's type
2. runs the handler
3. JSON-encodes its result
4. streams the result to the client as `tool-output-available`
5. stores the call and its result in the conversation
6. sends the result to the model as a user message, `Result of <name>: <output>`, and asks the
   model again

If decoding fails, or the handler or a generated tool returns an error, the result is
`{"error":"<message>"}`, and the model sees that instead.

A chat request makes at most four model calls. Tool calls in the fourth response are streamed to
the client but are not run.
