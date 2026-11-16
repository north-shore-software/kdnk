---
title: Tools
---

`kdnk.Tool(r, name, description, handler)` registers a tool you write yourself:

```go
type ShipInput struct {
	Feature  string  `json:"feature" jsonschema:"What is being shipped"`
	Audience float64 `json:"audience" jsonschema:"How many users it is aimed at, in thousands"`
}

kdnk.Tool(r, "ship_feature", "Ship a feature to users.",
	func(ctx context.Context, app core.App, s kdnk.Session, in ShipInput) (any, error) {
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
6. sends the call and its result back to the model through the provider's native tool-calling
   format, and asks the model again

If decoding fails, or the handler or a generated tool returns an error, the result is
`{"error":"<message>"}`, and the model sees that instead.

A chat request makes at most four model calls. The fourth is made without any tools, so the model
has to answer in text.
