# Todo Example

## Overview
A task management assistant that helps users record and list tasks prioritized by urgency and expected value.

## What It Demonstrates
- **Static System Prompt**: Demonstrates a fixed system prompt configured once in `SystemPrompt()`.
- **Tool Calling & Persistence**: Demonstrates tools (`add_todo`, `list_todos`) that read and write records to a PocketBase database collection (`todos`).
- **User-Scoped Data**: Uses `s.UserID` from `plugin.Session` to filter records per user across all conversations.
- **Basic Plugin Architecture**: Implements `plugin.Plugin` with `Collections()`, `Tools()`, and `ExecuteTool()`.
- **Guest Auth**: The page signs itself in on load against `POST /api/guest`, which mints a `users` record with a server-generated `guestId` and a separate random password. `guestId` is a unique identity field on the `users` collection, so no fake email address is involved. The credentials live in `localStorage`; `POST /api/guest/upgrade` converts the same record into an email account, keeping its id and therefore its todos and history.

## What It Does NOT Demonstrate
- **Dynamic System Prompts**: System prompt does not change or reflect live state between turns.
- **Per-Conversation State**: State is scoped globally to the user (`owner = s.UserID`), not tied to individual conversation IDs.
- **State Machine Rules**: Does not enforce multi-turn game or state transition logic inside tool execution.
- **Plugin Data Cleanup**: `pruneGuests` deletes guests idle for 30 days and cascades to their conversations and messages, but `todos.owner` is a plain text field, so a pruned guest leaves its todo rows behind. A plugin that owns user-scoped data has no hook for cleaning it up.
- **Durable Guest Sessions**: Clearing `localStorage` abandons the guest account permanently. Guest auth is for saving state before sign-up, not a substitute for it.
