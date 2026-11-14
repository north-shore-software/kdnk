---
title: LLM providers
---

## Setting the provider

An app has exactly one provider, set in `setup` with `r.Provider`. It answers every chat request
from every user:

```go
r.Provider(&gemini.Provider{Model: "gemini-3.6-flash"})
```

Choosing between providers is up to your app. The examples read `AGENT_SMITH_MODEL_SET`: `local`,
the default, uses `gemma4` on Ollama, and `cloud` uses `gemini-3.6-flash` on Gemini.
`./smith.sh setup` writes that choice to `examples/config.env`, and `./smith.sh run` sets the
variable.

## Gemini

`gemini.Provider`, from `github.com/north-shore-software/agent-smith/providers/gemini`, streams
from Gemini's `streamGenerateContent` endpoint.

- `Model` is required, for example `gemini-3.6-flash`.
- The API key is read from `GEMINI_API_KEY` on every request, so it can be changed without a
  restart. Get one from [Google AI Studio](https://aistudio.google.com/apikey).
- A request fails if the key or the model is empty, or if Gemini returns an error status.

Known limitation: if the model calls the same tool twice in one response, the second call is
dropped.

## Ollama

`ollama.OllamaProvider`, from `github.com/north-shore-software/agent-smith/providers/ollama`,
streams from a local or remote [Ollama](https://ollama.com/) server.

- `Model` is the Ollama model name. If it's empty, `gemma4` is used.
- The server URL comes from `OLLAMA_HOST`, read once at startup, and defaults to
  `http://localhost:11434`. It is also available as `ollama.Base`.
- Each request asks for the model's full context length, and keeps the model loaded for 60
  minutes.

Known limitation: an error status from Ollama, for example because the model isn't pulled, is
not detected. The reply is just empty.

The package also has helpers for managing models on the Ollama server:

- `PullModel(ctx, model, onProgress)` downloads a model.
- `PreloadModel(ctx, model)` loads a model into memory, so the first chat request doesn't wait
  for it.
- `GetModelContextLength(ctx, model)` returns the context length Ollama reports for a model, or
  4096 if it can't find out.

## Writing a provider

A provider implements `providers.Provider`, from
`github.com/north-shore-software/agent-smith/providers`:

```go
type Provider interface {
	ModelName() string
	Stream(ctx context.Context, messages []Message, tools []Tool) (*Completion, error)
}
```

`ModelName` is stored with each assistant message.

`Stream` gets the conversation and the tools the model may call:

- Each `Message` has a `Role`, which is `system`, `user` or `assistant`, and its text `Content`.
  Tool results arrive as `user` messages that start with `Result of <tool name>:`.
- Each `Tool` has a `Name`, a `Description`, and its arguments as a JSON Schema object in
  `Parameters`.

Return an error only if the stream can't be started. Otherwise return a `Completion`:

- `Events` is a channel of `StreamEvent`s that you close when the response ends.
- `Wait` blocks until the stream has ended, and returns token `Usage`, or nil.

Send these events:

| Kind | Fields | Meaning |
|---|---|---|
| `EventText` | `Text` | A chunk of the reply. |
| `EventToolStart` | `ToolCallID`, `ToolName` | The model has started a tool call. |
| `EventToolArgDelta` | `ToolCallID`, `Text` | A chunk of the call's argument JSON. Optional. |
| `EventToolEnd` | `ToolCallID`, `ToolName`, `Args` | The call is complete, with its full JSON arguments. |

When the channel closes, the server runs every tool call that got an `EventToolEnd`, in order.
`ToolCallID` only has to be unique
within the response. If `Usage.TokensPerSecond` is above zero, it is sent to the client as a
`data-inference_rate` event.
