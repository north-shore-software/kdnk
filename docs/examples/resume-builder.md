---
title: "Example: resume-builder"
---

## What it does

A chat assistant that builds the work-experience section of a resume, one job at a time. It asks
for anything missing and turns what the user did into short bullet points. The code is in
`examples/resume-builder`.

## The app

Like the [todo example](/examples/todo/), it is one collection and a system prompt:

```go
type Job struct {
	Company  string `json:"company" jsonschema:"The employer's name"`
	Title    string `json:"title" jsonschema:"The job title"`
	Location string `json:"location" jsonschema:"Where the job was, e.g. a city or Remote"`
	Start    string `json:"start" jsonschema:"When the job started, as YYYY-MM"`
	End      string `json:"end" jsonschema:"When the job ended, as YYYY-MM, or empty if it is the current job"`
	Summary  string `json:"summary" jsonschema:"What the user did and achieved there, as short bullet points"`
}

agentsmith.Collection[Job](r, "jobs")
```

The model gets `jobs_create`, `jobs_list`, `jobs_get`, `jobs_update` and `jobs_delete`. Jobs
belong to the user, across all their conversations.

Limits:

- Only work experience is stored. There is no education, skills or contact details, and no
  rendered resume.
- Start and end months are plain text. The `YYYY-MM` format is described to the model but not
  enforced.

## The web UI

`examples/resume-builder/web/index.html` is the same page as the
[todo example](/examples/todo/#the-web-ui).

## Running it

From the repository root:

```sh
./smith.sh setup
./smith.sh run resume-builder
```

`./smith.sh clear resume-builder` deletes the example's data.
