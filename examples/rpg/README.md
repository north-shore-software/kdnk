# RPG Example

## Overview
A turn-based startup simulation game where the LLM acts as game master narrating a run about building an AI application.

## What It Demonstrates
- **Dynamic Per-Turn System Prompt**: Demonstrates generating the system prompt dynamically on every turn in `SystemPrompt()` to inject current game state (week, credits, users, quality, status).
- **Per-Conversation Session State**: Uses both `s.UserID` and `s.ConversationID` from `plugin.Session` to track isolated game runs in a `game_state` collection.
- **State Machine & Rule Enforcement**: Enforces turn-based mechanics (burning credits, random rolls, win/loss conditions) directly inside `ExecuteTool()`.
- **LLM as Narrator**: Demonstrates using tool outputs as source of truth for the LLM to narrate game outcomes.
- **Guest Auth**: The page signs itself in on load against `POST /api/guest`, which mints a `users` record with a server-generated `guestId` and a separate random password. `guestId` is a unique identity field on the `users` collection, so no fake email address is involved. The credentials live in `localStorage`; `POST /api/guest/upgrade` converts the same record into an email account, keeping its id and therefore its run history.

## What It Does NOT Demonstrate
- **Static Data Collections**: Does not provide CRUD management for persistent end-user domain data (e.g. task lists or documents).
- **Multi-Collection Schemas**: Relies on a single state record per conversation rather than relational collections.
- **Plugin Data Cleanup**: `pruneGuests` deletes guests idle for 30 days and cascades to their conversations and messages, but `game_state` rows are keyed by plain text owner and are left behind. A plugin that owns user-scoped data has no hook for cleaning it up.
- **Durable Guest Sessions**: Clearing `localStorage` abandons the guest account permanently. Guest auth is for saving state before sign-up, not a substitute for it.
