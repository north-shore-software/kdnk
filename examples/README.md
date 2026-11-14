# Examples

**These examples do not build.** They use the old API (`server.New`, `plugin.Plugin`, and the
`llm` model-set and provider-factory registry). That API has moved under `internal/`, and this
directory is a separate Go module, so it can no longer import it. The examples will be ported to
the new `agentsmith.Run` API once that API can serve, and `smith.sh` cannot run them until then.

This directory is its own module (`examples/go.mod`) so that the examples can only use the
library's public API, the same as a real app. The `go.work` at the repo root ties the two modules
together for local development.
