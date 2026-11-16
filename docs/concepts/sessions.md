---
title: Sessions
---

Tool handlers and the system prompt function receive an `kdnk.Session`:

- `UserID` is the ID of the signed-in user's record in the `users` collection.
- `ConversationID` is the conversation ID the client sent with the chat request.

Collections are scoped by `UserID`. Nothing is scoped by `ConversationID` yet, so state you store
in a collection is shared by all of a user's conversations.
