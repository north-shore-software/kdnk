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

`Collection[Todo]` gives the model `todos_create`, `todos_list`, `todos_get`, `todos_update` and
`todos_delete`. `Singleton[Profile]` gives it `profile_get` and `profile_set`, which read and
replace the user's location. The location is set through chat only.

The three tools written by hand call the Open-Meteo APIs, which need no key:

- `find_location` looks up a place name and returns up to 5 matches with latitude, longitude and
  IANA timezone. The model copies one into `profile_set`.
- `get_time` returns the current date, time and weekday in a timezone, or in UTC if the timezone
  is empty or unknown.
- `get_weather` returns a daily forecast at a latitude and longitude: conditions, minimum and
  maximum temperature in °C, chance and amount of precipitation, and maximum wind in km/h. It
  covers today and the next 6 days only, so it cannot see further than 7 days ahead.

The system prompt names the todo tools, and tells the model that when the user asks what to do on
a given day it should read the location with `profile_get` (asking for it and saving it if
missing), call `get_time` and `get_weather`, and avoid outdoor tasks on bad-weather days.

Todos and the profile belong to the user, not to a conversation, so a new conversation sees the
same list and location.

## The web UI

`examples/todo/web/index.html` is a single page with no build step, embedded in the binary and
served with `r.Web`. It:

- signs in as a guest on first load, with `POST /api/guest`, and keeps the guest credentials and
  a conversation ID in `localStorage`
- reloads the conversation from `chat_conversation` and `chat_message` through PocketBase's
  records API
- sends messages to `POST /api/chat`, reads the event stream, and shows each tool call with its
  arguments and result
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
