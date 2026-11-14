# RPG Example

## Overview
A turn-based startup simulation game where the LLM acts as game master narrating a run about building an AI application.

## What It Demonstrates
- **Dynamic Per-Turn System Prompt**: `Registry.SystemPromptFunc` builds the system prompt on every turn from the current game state (week, credits, users, quality, status).
- **Per-User Game State**: Each user has one game, stored in a `game_state` collection declared with `agentsmith.Hidden`, so the model gets no tools for it and only Go code reads and writes it. A new conversation continues the same game.
- **Custom Tools**: `ship_feature` and `tune_model` are registered with `agentsmith.Tool`; their JSON Schemas come from the handlers' argument types.
- **State Machine & Rule Enforcement**: Enforces turn-based mechanics (burning credits, random rolls, win/loss conditions) inside the tool handlers.
- **LLM as Narrator**: Demonstrates using tool outputs as source of truth for the LLM to narrate game outcomes.
- **Guest Auth**: The page signs itself in on load against `POST /api/guest`, which mints a `users` record with a server-generated `guestId` and a separate random password. `guestId` is a unique identity field on the `users` collection, so no fake email address is involved. The credentials live in `localStorage`; `POST /api/guest/upgrade` converts the same record into an email account, keeping its id and therefore its run history.

## What It Does NOT Demonstrate
- **Static Data Collections**: Does not provide CRUD management for persistent end-user domain data (e.g. task lists or documents).
- **Multi-Collection Schemas**: Relies on a single state record per user rather than relational collections.
- **Starting Over**: There is no way to reset a game. Once a run is won or lost, it stays over.
- **Durable Guest Sessions**: Clearing `localStorage` abandons the guest account permanently. Guest auth is for saving state before sign-up, not a substitute for it.
