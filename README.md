<p align="center">
  <img src="docs/agent-smith.jpg" alt="Agent Smith" width="480">
</p>

<p align="center"><strong>You hear that, Mr. Anderson? That is the sound of inevitability!</strong></p>

> [!WARNING]
> 🚧 **Under active development.** 🚧
>
> Interfaces, schemas, and endpoints change without notice, and there is no migration path
> between versions. Not ready for production use.

# kdnk

A lightweight all-in-one backend for stateful, deterministic apps whose main interface is an LLM
chat. It handles chat persistence, authentication, and guest sessions — and provides a
mutable data store that the user reads and writes through the LLM.

What is in the box today:

- Chat persistence
- Auth and guest sessions, with a guest → email upgrade path
- Streaming chat endpoint for the Vercel AI SDK
- Integrated tool calling to access the mutable data store

Supports:

- Gemini or Ollama LLMs
- App-defined data schema

Built on [PocketBase](https://pocketbase.io/): one Go binary, SQLite, admin dashboard included.

## Getting started

Requires Go 1.24.1 or newer, and either a [Gemini API key](https://aistudio.google.com/apikey) or a
local [Ollama](https://ollama.com/).

```sh
./kdnk.sh setup   # specify API keys, or a local model
./kdnk.sh run     # build and serve one of the example projects
./kdnk.sh clear   # clear an example's local data
```

## License

AGPL-3.0. See [LICENSE](LICENSE).
