# Todo Example

## Overview
A task management assistant that helps users record and list tasks prioritized by urgency and expected value, and takes the weather at the user's location into account when suggesting what to do on a given day.

## What It Demonstrates
- **Static System Prompt**: Sets a fixed system prompt with `Registry.SystemPrompt`.
- **Generated Collection Tools**: `kdnk.Collection[Todo]` creates the `todos` collection and gives the model `todos_create`, `todos_list`, `todos_get`, `todos_update` and `todos_delete`, with arguments taken from the fields of `Todo`.
- **Per-User Singleton**: `kdnk.Singleton[Profile]` stores each user's location (place name, latitude, longitude and timezone) and gives the model `profile_get` and `profile_set`. The location is set through chat only.
- **Custom Tools**: `kdnk.Tool` adds `find_location` (place name to coordinates and timezone, via the Open-Meteo geocoding API), `get_time` (current date, time and weekday in a timezone, or UTC if it is empty or unknown) and `get_weather` (daily forecast from the Open-Meteo forecast API, in metric units). `get_weather` covers today and the next 6 days only; it cannot see further than 7 days ahead.
- **User-Scoped Data**: Each todo belongs to the user who created it, across all conversations, and a user only sees their own.
- **Model Selection**: `KDNK_MODEL_SET` picks the provider: `local` (the default) uses `gemma4` on Ollama, `cloud` uses `gemini-3.6-flash` on Gemini.
- **Web UI**: Serves the embedded `web/` directory with `Registry.Web`.
- **Guest Auth**: The page signs itself in on load against `POST /api/guest`, which mints a `users` record with a server-generated `guestId` and a separate random password. `guestId` is a unique identity field on the `users` collection, so no fake email address is involved. The credentials live in `localStorage`; `POST /api/guest/upgrade` converts the same record into an email account, keeping its id and therefore its todos and history.

## What It Does NOT Demonstrate
- **Dynamic System Prompts**: System prompt does not change or reflect live state between turns.
- **Per-Conversation State**: State is scoped globally to the user (`owner = s.UserID`), not tied to individual conversation IDs.
- **State Machine Rules**: Does not enforce multi-turn game or state transition logic inside tool execution.
- **Durable Guest Sessions**: Clearing `localStorage` abandons the guest account permanently. Guest auth is for saving state before sign-up, not a substitute for it.
