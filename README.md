<picture>
  <source media="(prefers-color-scheme: dark)" srcset="brand/kdnk-banner-dark.svg">
  <img alt="kdnk: Describe the app. kdnk does the plumbing." src="brand/kdnk-banner-light.svg">
</picture>

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

Requires Go 1.24.1 or newer, Node.js and [pnpm](https://pnpm.io/installation), and either a
[Gemini API key](https://aistudio.google.com/apikey) or a local [Ollama](https://ollama.com/).

```sh
./kdnk.sh setup   # check tools, install JavaScript dependencies, specify API keys or a local model
./kdnk.sh run     # build and serve one of the example projects
./kdnk.sh dev     # serve an example with a Vite dev server for its frontend
./kdnk.sh clear   # clear an example's local data
```

## License

AGPL-3.0. See [LICENSE](LICENSE).
