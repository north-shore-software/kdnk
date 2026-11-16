---
title: System prompt
---

`r.SystemPrompt(text)` sets a prompt sent at the start of every chat request.

`r.SystemPromptFunc(fn)` builds it instead, once per chat request, before the first model call.
Use it when the prompt depends on stored state:

```go
r.SystemPromptFunc(func(ctx context.Context, app core.App, s kdnk.Session) (string, error) {
	rec, err := states.Get(ctx, s, id)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("The player is in week %d.", rec.Data.Week), nil
})
```

An empty prompt sends no system message. An error from the function fails the request with HTTP
500. Set one or the other, once.
