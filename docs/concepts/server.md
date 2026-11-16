---
title: Server configuration
---

## Running the server

`kdnk.Run` runs the PocketBase command line. Start the server with `serve`:

```sh
./myapp serve --http=127.0.0.1:8090 --dir ./pb_data
```

`--dir` is where the SQLite database lives, `pb_data` next to the binary by default.

The `migrate` command is available too. kdnk's own tables are created by a migration that
runs automatically. Your collections' tables are not migrations: they are created or extended
each time the server starts serving, as described in
[Collections](/concepts/collections/).

## Routes

On top of PocketBase's own API, the server adds:

| Route | Auth | What it does |
|---|---|---|
| `POST /api/guest` | none | Creates a guest user. See [Guest users](#guest-users). |
| `POST /api/guest/upgrade` | user | Gives a guest an email and password. |
| `POST /api/chat` | user | Stores new messages and streams the model's reply. |
| `GET /{path...}` | none | The web UI, if one is set. See [Serving a web UI](#serving-a-web-ui). |

Authenticated routes take a PocketBase auth token in the `Authorization` header.

**`POST /api/chat`** takes a conversation ID and the messages in the conversation:

```json
{
  "id": "conversation-1",
  "messages": [
    { "id": "m1", "role": "user", "parts": [{ "type": "text", "text": "hi" }] }
  ]
}
```

The conversation is looked up by `id` among the user's conversations, and created if it is new.
Messages whose `id` is already stored are ignored, so a client can send the whole conversation
each time. The model sees the system prompt, then the stored history, then the new messages.

The reply is a server-sent event stream in the
[AI SDK UI message stream](https://ai-sdk.dev/docs/ai-sdk-ui/stream-protocol) format: `start`,
`text-start`, `text-delta` and `text-end`, `tool-input-start` and `tool-input-available`,
`tool-output-available`, `error`, `finish`, then `[DONE]`. When the provider reports a generation
rate, a `data-inference_rate` event carries it.

If the first model call can't be started, for example because the API key is missing, the
request fails with HTTP 500 and the message `Llm stream failed.`, before any event is sent. A
later model call that fails sends an `error` event and ends the stream.

**History.** Conversations and messages are stored in the `chat_conversation` and `chat_message`
collections. A user can list and view their own records in both through PocketBase's standard
`/api/collections/.../records` API, which is how the examples reload a conversation.

## Guest users

`POST /api/guest` creates a user with a random 24-character `guestId` and a random 40-character
password, and returns an auth token. The credentials are returned in `meta`:

```json
{ "token": "…", "record": { … }, "meta": { "guestId": "…", "password": "…" } }
```

A client keeps them to sign in again later, through PocketBase's
`POST /api/collections/users/auth-with-password` with `identity` set to the `guestId`.

`POST /api/guest/upgrade` takes `{"email": "…", "password": "…"}` from a signed-in guest, and
turns the same user record into a normal email account. The user's ID doesn't change, so their
records and conversations stay theirs. It fails with HTTP 400 for a user that isn't a guest.

Every day at 03:00, guests created more than 30 days ago who haven't sent a message in the last
30 days are deleted, with all their records and conversations.

## Rate limits

If PocketBase's rate limiting is off when the server starts, it is turned on with these rules:

| Label | Requests | Per |
|---|---|---|
| `POST /api/guest` | 5 | 300 s |
| `POST /api/chat` | 30 | 60 s |
| `*:auth` | 5 | 10 s |
| `*:create` | 20 | 5 s |

If rate limiting is already on, its rules are left alone. Change them in the admin dashboard.

## Serving a web UI

`r.Web(fsys)` serves the files in `fsys` for `GET` requests that no other route matches. Paths not
in `fsys` get `index.html`, so a single-page app's own routes work. The examples embed theirs:

```go
//go:embed web
var webAssets embed.FS

func setup(r *kdnk.Registry) error {
	web, err := fs.Sub(webAssets, "web")
	if err != nil {
		return err
	}
	r.Web(web)
	// …
}
```

Without `Web`, no web UI is served.
