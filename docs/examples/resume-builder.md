---
title: "Example: resume-builder"
---

## What it does

A chat assistant that builds a full audit of the user's working life (past roles, war stories,
skills and contact details) through conversation. When the user gives it a job description, it
writes a resume tailored to that job from the facts in the audit. The code is in
`examples/resume-builder`.

## The app

The app is one per-user singleton, five collections and a system prompt:

```go
type Profile struct {
	Name     string `json:"name" jsonschema:"The user's full name"`
	Headline string `json:"headline" jsonschema:"One line describing who they are professionally"`
	Email    string `json:"email" jsonschema:"Contact email"`
	Phone    string `json:"phone" jsonschema:"Contact phone number"`
	Location string `json:"location" jsonschema:"City and region, or Remote"`
	Links    string `json:"links" jsonschema:"Portfolio, LinkedIn, GitHub and similar URLs, one per line"`
}

type PastRole struct {
	Company  string `json:"company" jsonschema:"The employer's name"`
	Title    string `json:"title" jsonschema:"The job title"`
	Location string `json:"location" jsonschema:"Where the role was, e.g. a city or Remote"`
	Start    string `json:"start" jsonschema:"When the role started, as YYYY-MM"`
	End      string `json:"end" jsonschema:"When the role ended, as YYYY-MM, or empty if it is current"`
	Summary  string `json:"summary" jsonschema:"What the user did there, as short bullet points"`
}

type WarStory struct {
	RoleID    string `json:"role_id" jsonschema:"ID of the past_roles record this happened in, or empty"`
	Title     string `json:"title" jsonschema:"A short label for the story"`
	Situation string `json:"situation" jsonschema:"The context and the problem"`
	Action    string `json:"action" jsonschema:"What the user personally did"`
	Result    string `json:"result" jsonschema:"What changed because of it, with numbers where possible"`
}

type Skill struct {
	Name     string `json:"name" jsonschema:"The skill"`
	Category string `json:"category" jsonschema:"e.g. language, tool, domain, leadership"`
	Evidence string `json:"evidence" jsonschema:"Where the user showed it, naming roles or war stories"`
}

type JobDescription struct {
	Company string `json:"company" jsonschema:"The hiring company"`
	Title   string `json:"title" jsonschema:"The job title"`
	Text    string `json:"text" jsonschema:"The full job description as the user gave it"`
}

type Resume struct {
	JobDescriptionID string `json:"job_description_id" jsonschema:"ID of the job_descriptions record it targets"`
	Content          string `json:"content" jsonschema:"The resume, as markdown"`
}

func setup(r *kdnk.Registry) error {
	// provider() and web are set up above; see main.go
	r.Provider(p)
	r.SystemPrompt("You help the user build a complete audit of their working life, …")
	r.Web(web)
	kdnk.Singleton[Profile](r, "profile")
	kdnk.Collection[PastRole](r, "past_roles")
	kdnk.Collection[WarStory](r, "war_stories")
	kdnk.Collection[Skill](r, "skills")
	kdnk.Collection[JobDescription](r, "job_descriptions")
	kdnk.Collection[Resume](r, "resumes", kdnk.AppendOnly)
	return nil
}
```

`provider()` picks the model from `KDNK_MODEL_SET`: `gemma4` on Ollama if it is empty or `local`,
`gemini-3.6-flash` on Gemini if it is `cloud`. Any other value stops the app with an error.

`past_roles`, `war_stories`, `skills` and `job_descriptions` each give the model `_create`,
`_list`, `_get`, `_update` and `_delete` tools. `resumes` is append-only, so the model gets only
`resumes_create`, `resumes_list` and `resumes_get`: every resume is a new record, and none is
edited or deleted. `Singleton[Profile]` gives it `profile_get` and `profile_set`.

The system prompt tells the model to record roles, war stories (situation, action, result),
skills with evidence, and contact details; to ask follow-up questions for concrete results and
never invent facts; to save any job description the user gives; and, to write a resume, to read
the profile, list every past role, war story and skill, write a markdown resume tailored to the
job description from recorded facts only, and save it as a new record in `resumes`.

There are no custom tools. The audit and the resumes belong to the user, not to a conversation,
so a new conversation sees the same records.

Limits:

- `war_stories.role_id` and `resumes.job_description_id` are plain strings. Nothing checks that
  they hold the ID of an existing record, and deleting a role or job description leaves them
  pointing at nothing.
- Start and end months are plain text. The `YYYY-MM` format is described to the model but not
  enforced.
- Nothing checks that the model read the whole audit before writing a resume.
- Resumes are markdown shown only as tool results in the chat; there is no rendered resume.

## The web UI

`examples/resume-builder/web` is a Vite project using TypeScript and React. It does what the
[todo example's page](/examples/todo/#the-web-ui) does, with a terminal-style theme:

- signs in as a guest on first load, with `POST /api/guest`, and keeps the guest credentials and
  a conversation ID in `localStorage`
- reloads the conversation from `chat_conversation` and `chat_message` through PocketBase's
  records API, or shows an opening message if there is none
- sends messages to `POST /api/chat`, reads the event stream, and shows each tool call with its
  arguments and result
- signs in again and retries once, as the stored guest or a new one, if loading the conversation
  or sending a message is refused with 401
- has a form to save the guest account with an email and password, through
  `POST /api/guest/upgrade`, and to sign in to an existing account

The code is split by job:

- `src/api.ts` wraps the HTTP calls: guest and password sign-in, account upgrade, loading the
  history, and streaming a chat reply. Each throws `HttpError` on a non-2xx response.
- `src/useSession.ts` holds the token and guest credentials and signs in, again, or as a guest.
- `src/useChat.ts` holds the conversation ID and the log, loads the history and sends messages.
- `App.tsx`, `Log.tsx`, `ChatForm.tsx` and `AccountForm.tsx` render the page. `index.html` is the
  Vite entry and holds the styles.

Clearing `localStorage` abandons the guest account.

`pnpm build` writes the page to `web/dist/app`, and `main.go` embeds `web/dist` and serves
`web/dist/app`. Only `web/dist/.gitkeep` is committed, so a Go build that skips `pnpm build`
compiles but serves nothing at `/`.

## Running it

From the repository root:

```sh
./kdnk.sh setup
./kdnk.sh run resume-builder
```

`run` builds the web app with `pnpm build`, then builds and starts the Go server. Open the page and
try `hi, I'd like to start my career audit`. `./kdnk.sh clear resume-builder` deletes the
example's data.

To work on the frontend, use `./kdnk.sh dev resume-builder` instead. It starts the Go server on
the configured port and a Vite dev server that forwards `/api` to it. Open the URL Vite prints.
