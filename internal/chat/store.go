package chat

import (
	"context"
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	// CollectionConversation is the name of the PocketBase collection holding conversations.
	CollectionConversation = "chat_conversation"
	// CollectionMessage is the name of the PocketBase collection holding messages.
	CollectionMessage = "chat_message"
)

// FindOrCreateConversation returns the conversation owned by ownerID whose
// external_conversation_id is clientID, creating it if none exists. If creation fails and the
// conversation now exists, the existing record is returned. ctx is unused.
func FindOrCreateConversation(ctx context.Context, app core.App, ownerID, clientID string) (*core.Record, error) {
	params := dbx.Params{"owner": ownerID, "cid": clientID}
	const filter = "owner = {:owner} && external_conversation_id = {:cid}"

	rec, err := app.FindFirstRecordByFilter(CollectionConversation, filter, params)
	if err == nil {
		return rec, nil
	}

	col, err := app.FindCollectionByNameOrId(CollectionConversation)
	if err != nil {
		return nil, err
	}
	rec = core.NewRecord(col)
	rec.Set("owner", ownerID)
	rec.Set("external_conversation_id", clientID)
	if err := app.Save(rec); err != nil {
		if existing, e2 := app.FindFirstRecordByFilter(CollectionConversation, filter, params); e2 == nil {
			return existing, nil
		}
		return nil, err
	}
	return rec, nil
}

// LoadMessages returns the messages of conversation ordered by seq. Records whose content does not
// decode as a Message are skipped. ctx is unused.
func LoadMessages(ctx context.Context, app core.App, conversation *core.Record) ([]Message, error) {
	recs, err := app.FindRecordsByFilter(
		CollectionMessage,
		"conversation = {:cid}",
		"seq", 0, 0,
		dbx.Params{"cid": conversation.Id},
	)
	if err != nil {
		return nil, err
	}

	msgs := make([]Message, 0, len(recs))
	for _, r := range recs {
		var m Message
		if err := json.Unmarshal([]byte(r.GetString("content")), &m); err == nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

// PersistMessage saves msg as a new message in conversation with the given seq and model name.
// model may be empty. ctx is unused.
func PersistMessage(ctx context.Context, app core.App, conversation *core.Record, msg Message, seq int, model string) error {
	col, err := app.FindCollectionByNameOrId(CollectionMessage)
	if err != nil {
		return err
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	rec := core.NewRecord(col)
	rec.Set("conversation", conversation.Id)
	rec.Set("seq", seq)
	rec.Set("content", types.JSONRaw(b))
	rec.Set("model", model)
	return app.Save(rec)
}
