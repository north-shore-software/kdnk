---
title: The setup function
---

An app is a function that takes an `*kdnk.Registry`. Pass it to `kdnk.Run` from
`main`:

```go
func main() {
	if err := kdnk.Run(setup); err != nil {
		log.Fatal(err)
	}
}

func setup(r *kdnk.Registry) error {
	r.Provider(&providers.Gemini{Model: "gemini-3.6-flash"})
	r.SystemPrompt("You are a note-taking assistant.")
	kdnk.Collection[Note](r, "notes")
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
