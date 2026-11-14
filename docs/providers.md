---
title: LLM providers
---

## Setting the provider

An app has exactly one provider, set in `setup` with `r.Provider`. It answers every chat request
from every user. It is either `providers.Gemini` or `providers.Ollama`, from
`github.com/north-shore-software/kdnk/providers`:

```go
r.Provider(&providers.Gemini{Model: "gemini-3.6-flash"})
```

Choosing between providers is up to your app. The examples read `KDNK_MODEL_SET`: `local`,
the default, uses `gemma4` on Ollama, and `cloud` uses `gemini-3.6-flash` on Gemini.
`./kdnk.sh setup` writes that choice to `examples/config.env`, and `./kdnk.sh run` sets the
variable.

## Gemini

`providers.Gemini` streams from Gemini's `streamGenerateContent` endpoint.

- `Model` is required, for example `gemini-3.6-flash`.
- The API key is read from `GEMINI_API_KEY` on every request, so it can be changed without a
  restart. Get one from [Google AI Studio](https://aistudio.google.com/apikey).
- A request fails if the key or the model is empty, or if Gemini returns an error status.

Known limitation: if the model calls the same tool twice in one response, the second call is
dropped.

## Ollama

`providers.Ollama` streams from a local or remote [Ollama](https://ollama.com/) server.

- `Model` is the Ollama model name. If it's empty, `gemma4` is used.
- The server URL comes from `OLLAMA_HOST`, read once at startup, and defaults to
  `http://localhost:11434`. It is also available as `providers.OllamaHost`.
- Each request asks for the model's full context length, and keeps the model loaded for 60
  minutes.

Known limitation: an error status from Ollama, for example because the model isn't pulled, is
not detected. The reply is just empty.

The package also has helpers for managing models on the Ollama server:

- `PullOllamaModel(ctx, model, onProgress)` downloads a model.
- `PreloadOllamaModel(ctx, model)` loads a model into memory, so the first chat request doesn't
  wait for it.
- `OllamaContextLength(ctx, model)` returns the context length Ollama reports for a model, or
  4096 if it can't find out.
