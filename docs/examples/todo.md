---
title: "Example: todo"
---

## What it does

A chat assistant that keeps a task list and helps the user prioritise it by urgency and expected
value, both on a 1–10 scale. When the user asks what to do on a given day, it checks the weather at
the user's location and avoids suggesting outdoor tasks on bad-weather days. The code is in
`examples/todo`.

## The app

The app is one collection, one per-user singleton, three custom tools and a system prompt:

```go
type Todo struct {
	Title         string  `json:"title" jsonschema:"What the task is"`
	Urgency       float64 `json:"urgency" jsonschema:"1-10, how soon it must happen"`
	ExpectedValue float64 `json:"expected_value" jsonschema:"1-10, how much it is worth"`
}

type Profile struct {
	Location  string  `json:"location" jsonschema:"Place name as the user gave it"`
	Latitude  float64 `json:"latitude" jsonschema:"From find_location"`
	Longitude float64 `json:"longitude" jsonschema:"From find_location"`
	Timezone  string  `json:"timezone" jsonschema:"IANA timezone from find_location"`
}

func setup(r *kdnk.Registry) error {
	// provider() and web are set up above; see main.go
	r.Provider(p)
	r.SystemPrompt("You help the user prioritise tasks by expected value and urgency, …")
	r.Web(web)
	kdnk.Collection[Todo](r, "todos")
	kdnk.Singleton[Profile](r, "profile")
	kdnk.Tool(r, "find_location", "Look up a place by name. …", findLocation)
	kdnk.Tool(r, "get_time", "Get the current date, time and weekday …", getTime)
	kdnk.Tool(r, "get_weather", "Get the daily weather forecast …", getWeather)
	return nil
}
```

`provider()` picks the model from `KDNK_MODEL_SET`: `gemma4` on Ollama if it is empty or `local`,
`gemini-3.6-flash` on Gemini if it is `cloud`. Any other value stops the app with an error.

`Collection[Todo]` gives the model `todos_create`, `todos_list`, `todos_get`, `todos_update` and
`todos_delete`. `Singleton[Profile]` gives it `profile_get` and `profile_set`, which read and
replace the user's location. `profile_set` must be passed the current `version`, so the model
calls `profile_get` before saving. The location is set through chat only.

The three tools written by hand call the Open-Meteo APIs, which need no key. Each request times out
after 10 seconds, and a network failure or a non-200 status is returned as an error, which the
model sees as `{"error":"<message>"}`.

- `find_location` looks up a place name and returns up to 5 matches, each with `name`, `region`,
  `country`, `latitude`, `longitude` and IANA `timezone`. The model copies one into
  `profile_set`. If nothing matches, it returns the error `no place found for "<name>"`.
- `get_time` returns the current `time` (RFC 3339), `date`, `weekday` and `timezone` in an IANA
  timezone. If the timezone is empty or cannot be loaded, it uses UTC without reporting an
  error, and `timezone` is `UTC`. The binary embeds Go's timezone database (`time/tzdata`), so
  this does not depend on the host's timezone files.
- `get_weather` returns the location's `timezone` and a list of `days`, dated in that timezone.
  Each day has `date`, `weekday`, `conditions`, `temp_max_c`, `temp_min_c`,
  `precipitation_probability_pct`, `precipitation_mm` and `wind_max_kmh`. It covers today and
  the next 6 days only, so it cannot see further than 7 days ahead. `conditions` is one of
  clear, partly cloudy, overcast, fog, drizzle, rain, snow, rain showers, snow showers or
  thunderstorm, or `weather code <n>` for any other WMO weather code. If the forecast's daily
  fields have different lengths, it returns the error `malformed forecast`.

The system prompt names the todo tools, and tells the model that when the user asks what to do on
a given day it should read the location with `profile_get` (asking for it and saving it with
`find_location` and `profile_set` if missing), call `get_time` and `get_weather`, and avoid
outdoor tasks on bad-weather days. It also says `get_weather` only covers the next 7 days, and
that after a tool returns the model should tell the user what happened in plain language.

Todos and the profile belong to the user, not to a conversation, so a new conversation sees the
same list and location.

## The web UI

`examples/todo/web/index.html` is a single page with no build step, embedded in the binary with
its icon, `favicon.svg`, and served with `r.Web`. It:

- signs in as a guest on first load, with `POST /api/guest`, and keeps the guest credentials and
  a conversation ID in `localStorage`
- reloads the conversation from `chat_conversation` and `chat_message` through PocketBase's
  records API
- sends messages to `POST /api/chat`, reads the event stream, and shows each tool call with its
  arguments and result
- signs in again and retries, as the stored guest or a new one, if loading the conversation or
  sending a message is refused with 401
- has a form to save the guest account with an email and password, through
  `POST /api/guest/upgrade`, and to sign in to an existing account

Clearing `localStorage` abandons the guest account.

## Running it

From the repository root:

```sh
./kdnk.sh setup      # pick Gemini (needs a key) or Ollama, and a port
./kdnk.sh run todo
```

Then open `http://127.0.0.1:8090/` (or the port you chose) and try
`add: file taxes, urgency 8, value 9`, then `I live in Calgary` and `what should I do this
weekend?`. `./kdnk.sh clear todo` deletes the example's data.

`./kdnk.sh run todo --dev` (or `run --dev todo`) starts PocketBase with its `--dev` flag, which
prints logs and every SQL statement to the console.
