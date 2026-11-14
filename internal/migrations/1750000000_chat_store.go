package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/north-shore-software/agent-smith/internal/chat"
)

const (
	guestIDField    = "guestId"
	guestIDIndex    = "idx_users_guestId"
	usersCreateRule = "isGuest = false"
	usersUpdateRule = "id = @request.auth.id" +
		" && @request.body.isGuest:isset = false" +
		" && @request.body.guestId:isset = false"
)

type indexDef struct {
	Name    string
	Unique  bool
	Columns string
}

type tableDef struct {
	Name       string
	ListRule   string
	ViewRule   string
	DeleteRule string
	Fields     []core.Field
	Indexes    []indexDef
}

var schema = []tableDef{
	{
		Name:       chat.CollectionConversation,
		ListRule:   "owner = @request.auth.id",
		ViewRule:   "owner = @request.auth.id",
		DeleteRule: "owner = @request.auth.id",
		Fields: []core.Field{
			&core.RelationField{Name: "owner", CollectionId: "users", Required: true, MaxSelect: 1, CascadeDelete: true},
			&core.TextField{Name: "external_conversation_id", Required: true},
			&core.AutodateField{Name: "created", OnCreate: true},
		},
		Indexes: []indexDef{
			{Name: "idx_chat_conversation_owner_external", Unique: true, Columns: "owner, external_conversation_id"},
		},
	},
	{
		Name:     chat.CollectionMessage,
		ListRule: "conversation.owner = @request.auth.id",
		ViewRule: "conversation.owner = @request.auth.id",
		Fields: []core.Field{
			&core.RelationField{Name: "conversation", CollectionId: chat.CollectionConversation, Required: true, MaxSelect: 1, CascadeDelete: true},
			&core.NumberField{Name: "seq"},
			&core.JSONField{Name: "content", MaxSize: 5 << 20},
			&core.TextField{Name: "model"},
			&core.AutodateField{Name: "created", OnCreate: true},
		},
		Indexes: []indexDef{
			{Name: "idx_chat_message_conversation_seq", Columns: "conversation, seq"},
		},
	},
}

func ensureField(c *core.Collection, f core.Field) {
	if c.Fields.GetByName(f.GetName()) == nil {
		c.Fields.Add(f)
	}
}

func rule(v string) *string {
	if v == "" {
		return nil
	}
	return types.Pointer(v)
}

func ensureCollection(app core.App, def tableDef) error {
	c, err := app.FindCollectionByNameOrId(def.Name)
	if err != nil {
		c = core.NewBaseCollection(def.Name)
	}

	c.Type = core.CollectionTypeBase
	c.ListRule = rule(def.ListRule)
	c.ViewRule = rule(def.ViewRule)
	c.CreateRule = nil
	c.UpdateRule = nil
	c.DeleteRule = rule(def.DeleteRule)

	for _, f := range def.Fields {
		if relField, ok := f.(*core.RelationField); ok {
			target, err := app.FindCollectionByNameOrId(relField.CollectionId)
			if err != nil {
				return err
			}
			relField.CollectionId = target.Id
		}
		ensureField(c, f)
	}
	for _, idx := range def.Indexes {
		c.AddIndex(idx.Name, idx.Unique, idx.Columns, "")
	}
	return app.Save(c)
}

func addIdentityField(c *core.Collection, name string) {
	for _, f := range c.PasswordAuth.IdentityFields {
		if f == name {
			return
		}
	}
	c.PasswordAuth.IdentityFields = append(c.PasswordAuth.IdentityFields, name)
}

func removeIdentityField(c *core.Collection, name string) {
	kept := make([]string, 0, len(c.PasswordAuth.IdentityFields))
	for _, f := range c.PasswordAuth.IdentityFields {
		if f != name {
			kept = append(kept, f)
		}
	}
	c.PasswordAuth.IdentityFields = kept
}

func init() {
	m.Register(func(app core.App) error {
		usersCol, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		ensureField(usersCol, &core.TextField{Name: guestIDField})
		ensureField(usersCol, &core.BoolField{Name: "isGuest"})

		if ef, ok := usersCol.Fields.GetByName("email").(*core.EmailField); ok {
			ef.Required = false
		}

		usersCol.AddIndex(guestIDIndex, true, guestIDField, guestIDField+" != ''")
		addIdentityField(usersCol, guestIDField)

		usersCol.CreateRule = types.Pointer(usersCreateRule)
		usersCol.UpdateRule = types.Pointer(usersUpdateRule)

		if err := app.Save(usersCol); err != nil {
			return err
		}

		for _, t := range schema {
			if _, err := app.FindCollectionByNameOrId(t.Name); err != nil {
				if err := app.Save(core.NewBaseCollection(t.Name)); err != nil {
					return err
				}
			}
		}
		for _, t := range schema {
			if err := ensureCollection(app, t); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for i := len(schema) - 1; i >= 0; i-- {
			c, err := app.FindCollectionByNameOrId(schema[i].Name)
			if err != nil {
				continue
			}
			if err := app.Delete(c); err != nil {
				return err
			}
		}

		usersCol, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}

		removeIdentityField(usersCol, guestIDField)
		usersCol.RemoveIndex(guestIDIndex)
		usersCol.Fields.RemoveByName(guestIDField)
		usersCol.Fields.RemoveByName("isGuest")

		if ef, ok := usersCol.Fields.GetByName("email").(*core.EmailField); ok {
			ef.Required = true
		}

		usersCol.CreateRule = types.Pointer("")
		usersCol.UpdateRule = types.Pointer("id = @request.auth.id")

		return app.Save(usersCol)
	})
}
