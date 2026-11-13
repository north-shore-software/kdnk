# Implementation plan: new API direction

Written 2026-11-13. This is a proposal. Nothing in it is approved until the user says so, and
that includes the order of the steps.

It implements the design in [api-direction.md](api-direction.md); read that first. The target
is `docs/quickstart.md`.

## Ground rules

- **Each step is gated.** Each step is at most one public API surface (the AGENTS.md
  implementation ceiling). For each one:
  1. Propose the signatures, types, names and error cases as text.
  2. Wait for sign-off.
  3. Implement.
  4. Report.
- **No step starts while one of its blocking decisions (below) is open.**
- **The old API stays working until step 6.** The new root package is built as a layer over the
  existing internals, so the examples keep running and each step can be checked against a real
  server.
- **Verification for each step:**
  - `go build ./...` and `go vet ./...`.
  - Compile the quickstart's code in a scratch module that has a `replace` pointing at this repo.
  - Run it against a live server where the step allows.
  - A real model reply needs a key, and this VM has none, so the user must run that check.
  - No tests unless the user asks for them.

## Decisions needed

Each item names the step it blocks. The ones marked *(open)* are also listed in
api-direction.md. The ones marked *(new)* came up while writing this plan.

| # | Decision | Blocks |
|---|---|---|
| D1 | Root package name, and what it contains *(open)* | 1 |
| D2 | What happens on errors in setup: duplicate names, unsupported struct fields, an error returned from `setup` *(open)* | 1 |
| D3 | Where server options go now that `server.Config` is gone: the embedded web UI (`Web fs.FS`, used by every example) and `HideStartBanner`. For example `r.Web(fs)`. *(new)* | 1 |
| D4 | Handler signature: what a handler gets instead of `core.App` *(open, next to shape)* | 2 |
| D5 | Schema generation: take a dependency or write our own *(open)* | 2 |
| D6 | Which Go field types a collection struct may use, and which PocketBase field each one maps to *(new)* | 3 |
| D7 | Generated tool names, and the arguments and results of each CRUD tool *(open)* | 3 |
| D8 | List order and the default limit *(open)* | 3 |
| D9 | Restriction options: names and syntax *(open)* | 3 |
| D10 | What happens when a collection's struct changes after its table exists. Today `ensureCollections` only creates missing tables and never alters them. *(new)* | 3 |
| D11 | Typed collection access from Go, e.g. `notes.List(ctx, s)` *(depends on D4)* | 3 |
| D12 | Provider: one registered provider for now, or design per-conversation choice first *(open)* | 4 |
| D13 | The `chat` subcommand: HTTP or in-process, persisting the guest, one-shot mode *(open)* | 5 |
| D14 | What moves to `internal/`. This needs explicit approval. *(open)* | 6 |
| D15 | What stays public for custom providers. They need `llm.Provider` and its request and event types; do they stay in `llm` or move to the root package? *(new)* | 6 |

## Steps

### 1. Root package skeleton

- **Adds:** a new package at the repo root containing `Registry`, `Run`, `Session` and the server
  options from D3.
- **How:** `Run` calls `setup`, freezes the registry, and builds the app with the existing
  `server` package. The registry is converted internally into whatever the server consumes today.
- **Visible result:** nothing yet beyond today's behaviour. Registering only a system prompt
  should serve chat.

### 2. Typed tools

- **Adds:** `agentsmith.Tool[In](r, name, description, handler)`.
- **How it behaves:**
  - The JSON Schema is generated from `In`, including `jsonschema` tags.
  - When the model calls a tool, its arguments are decoded into `In`.
  - The handler's `any` result is JSON-encoded.
  - Tool calls are dispatched by name, internally, replacing `ExecuteTool`.
- **Check:** the quickstart's `get_time` works.

### 3. Collections

This is the largest step and may need splitting, for example 3a declaration and storage, 3b
generated tools, 3c typed Go access.

- **Adds:** `agentsmith.Collection[T](r, name, options...)`.
- **How it behaves:**
  - The struct is mapped to PocketBase fields (D6).
  - Every collection gets a hidden owner field, plus a hidden created field if D8 needs it.
  - Tables are created at serve time (D10).
  - The CRUD tools (D7) are generated, and every query is scoped to the session's user.
  - Restrictions (D9) are enforced on the generated tools and nowhere else: app code can still
    write.
  - List uses a limit (D8).
  - Typed Go access (D11) is included.
- **Check:** the quickstart's notes flow. Save a note, list it, update it, delete it. Then confirm
  a second guest can't see the first guest's notes.

### 4. Provider and system prompt

- **Adds:** `r.Provider` and `r.SystemPrompt`.
- **Removes from the app's path:** the global `llm` model-set and provider-factory registry
  (`SetActiveModelSet`, `SetProviderFactory`, `EnsureReady`, model names hard-coded in `llm`).
- **Fixes friction:** a missing or invalid API key should show a clear error on the server's
  output, not just "Llm stream failed.". How it should look is a small decision in its own right.
- **Check:** the quickstart's `main` and `setup` compile unchanged.

### 5. `chat` subcommand

- **Adds:** a `chat` command registered by `Run` (D13).
- **Implementation note:** it consumes the AI SDK UI message stream: it prints text deltas, and
  prints tool calls as `[name]`.
- **Check:** the quickstart's sample session, run by the user with a key.

### 6. Shrink the public surface

- **Does:** moves everything that isn't app-facing under `internal/` (D14, D15).
- **Candidates:** `server`, `chat`, `migrations`, `httpx`, the `llm` registry and selector, and
  the `plugin` package (deleted).
- **Stays public:** the root package, `gemini`, `ollama`, and whatever custom providers need.
- **Side effect:** this also fixes the AGENTS.md conflict over `httpx` doc comments.
- **Ordering constraint:** this breaks the old examples, so step 7 must happen in the same change
  or immediately before.

### 7. Port the examples and tooling

- **todo:** a collection plus restrictions where they fit, and the web UI through D3.
- **rpg:**
  - Its `game_state` is per-conversation data, and per-user is the only scope so far. Port it as
    well as the current design allows, and record what's missing. It's the use case that will
    drive scopes beyond per-user.
  - It may also need a dynamic system prompt; check the code.
- **resume-builder:** still a copy of todo. Either port it as-is, or ask what it should be.
- **`smith.sh`:** it uses `AGENT_SMITH_MODEL_SET` and the cloud/local model sets, which step 4
  removes. It needs a replacement way to pick the provider, which is a decision.
- **`README.md`:** describes the old setup; update it.

### 8. Docs

- **Quickstart:** becomes descriptive again. Verify it end to end and fix anything that drifted.
- **Remove the exception:** take the prescriptive-quickstart note out of api-direction.md, or
  retire that file.
- **Pages:**
  - `docs/plugins.md` gets renamed, because there is no plugin type any more.
  - Fill or update `server.md` and `providers.md` for what changed. Content beyond that is a
    separate task.
- **Reference:** update the package list in `docs/site/scripts/reference.mjs`, then check
  `pnpm build`.
- **Doc comments:** every new exported identifier gets one (AGENTS.md).

## Known bugs this work makes worse

The user said to fix these later. Two of them become more likely once CRUD tools are generated,
so the user may want them scheduled before or alongside step 3:

- `gemini/gemini.go:197-224`: a second call to the same tool in one turn is dropped. With
  generated tools, this is routine. "Save these three notes" means three `notes_create` calls.
- `server/chat.go:106`: when the tool-step limit (4 model calls) is hit, tool calls are left
  without output. CRUD-heavy turns will hit the limit more often.

The rest of the list in api-direction.md is unaffected by this plan.
