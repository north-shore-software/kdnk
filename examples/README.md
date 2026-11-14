# Examples

This directory is its own module (`examples/go.mod`) so that the examples can only use the
library's public API, the same as a real app. The `go.work` at the repo root ties the two modules
together for local development.
