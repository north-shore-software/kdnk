package kdnk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/kdnk/providers"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

type tool struct {
	def providers.Tool
	run func(ctx context.Context, app core.App, s Session, args json.RawMessage) (any, error)
}

type idArgs struct {
	ID string `json:"id"`
}

type versionArgs struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

var idProperty = map[string]any{"type": "string", "description": "The record's ID."}

var versionProperty = map[string]any{"type": "string", "description": "The record's version, as last returned for it."}

func versionOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

func (c *Store[T]) ifVersion(ctx context.Context, s Session, in versionArgs, write func(tx core.App, rec *core.Record) (any, error)) (any, error) {
	if c.app == nil {
		return nil, errUnbound
	}
	var out any
	err := c.app.RunInTransaction(func(tx core.App) error {
		rec, err := c.find(ctx, tx, s, in.ID)
		if err != nil {
			return err
		}
		cur := c.read(rec)
		data, err := json.Marshal(cur.Data)
		if err != nil {
			return err
		}
		if versionOf(data) != in.Version {
			b, err := c.toolRecord(cur)
			if err != nil {
				return err
			}
			out = map[string]any{"error": c.name + " record was modified since it was read", "current": b}
			return nil
		}
		out, err = write(tx, rec)
		return err
	})
	return out, err
}

func (c *Store[T]) tools() []tool {
	switch c.restriction {
	case appendOnly:
		return []tool{c.createTool(), c.listTool(), c.getTool()}
	case readOnly:
		return []tool{c.listTool(), c.getTool()}
	case hidden:
		return nil
	}
	return []tool{c.createTool(), c.listTool(), c.getTool(), c.updateTool(), c.deleteTool()}
}

func (c *Store[T]) properties() (map[string]any, []string) {
	return fieldProperties(c.fields)
}

func fieldProperties(fields []field) (map[string]any, []string) {
	props := map[string]any{}
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		p := map[string]any{}
		switch f.kind {
		case kindText:
			p["type"] = "string"
		case kindBool:
			p["type"] = "boolean"
		case kindInt, kindUint:
			p["type"] = "integer"
		case kindFloat:
			p["type"] = "number"
		case kindDate:
			p["type"] = "string"
			p["format"] = "date-time"
		}
		if f.desc != "" {
			p["description"] = f.desc
		}
		props[f.name] = p
		names = append(names, f.name)
	}
	return props, names
}

func objectSchema(props map[string]any, required []string) json.RawMessage {
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	b, _ := json.Marshal(s)
	return b
}

func decodeArgs(args json.RawMessage, v any) error {
	if len(args) == 0 {
		return nil
	}
	return json.Unmarshal(args, v)
}

func (c *Store[T]) toolRecord(r Record[T]) (json.RawMessage, error) {
	data, err := json.Marshal(r.Data)
	if err != nil {
		return nil, err
	}
	m := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m["id"], err = json.Marshal(r.ID); err != nil {
		return nil, err
	}
	if m["created"], err = json.Marshal(r.Created); err != nil {
		return nil, err
	}
	if m["version"], err = json.Marshal(versionOf(data)); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func (c *Store[T]) createTool() tool {
	props, _ := c.properties()
	return tool{
		def: providers.Tool{
			Name:        c.name + "_create",
			Description: "Create a record in " + c.name + ".",
			Parameters:  objectSchema(props, nil),
		},
		run: func(ctx context.Context, _ core.App, s Session, args json.RawMessage) (any, error) {
			var v T
			if err := decodeArgs(args, &v); err != nil {
				return nil, err
			}
			r, err := c.Create(ctx, s, v)
			if err != nil {
				return nil, err
			}
			return c.toolRecord(r)
		},
	}
}

func (c *Store[T]) listTool() tool {
	props := map[string]any{"limit": map[string]any{
		"type":        "integer",
		"description": "The maximum number of records to return, from 1 to 100. Defaults to 20.",
	}}
	return tool{
		def: providers.Tool{
			Name:        c.name + "_list",
			Description: "List records in " + c.name + ", newest first.",
			Parameters:  objectSchema(props, nil),
		},
		run: func(ctx context.Context, _ core.App, s Session, args json.RawMessage) (any, error) {
			var in struct {
				Limit *float64 `json:"limit"`
			}
			if err := decodeArgs(args, &in); err != nil {
				return nil, err
			}
			limit := defaultListLimit
			if in.Limit != nil && *in.Limit >= 1 {
				limit = int(min(*in.Limit, maxListLimit))
			}
			rs, err := c.List(ctx, s, limit)
			if err != nil {
				return nil, err
			}
			out := make([]json.RawMessage, 0, len(rs))
			for _, r := range rs {
				b, err := c.toolRecord(r)
				if err != nil {
					return nil, err
				}
				out = append(out, b)
			}
			return out, nil
		},
	}
}

func (c *Store[T]) getTool() tool {
	return tool{
		def: providers.Tool{
			Name:        c.name + "_get",
			Description: "Get a record in " + c.name + " by ID.",
			Parameters:  objectSchema(map[string]any{"id": idProperty}, []string{"id"}),
		},
		run: func(ctx context.Context, _ core.App, s Session, args json.RawMessage) (any, error) {
			var in idArgs
			if err := decodeArgs(args, &in); err != nil {
				return nil, err
			}
			r, err := c.Get(ctx, s, in.ID)
			if err != nil {
				return nil, err
			}
			return c.toolRecord(r)
		},
	}
}

func (c *Store[T]) updateTool() tool {
	props, names := c.properties()
	props["id"] = idProperty
	props["version"] = versionProperty
	return tool{
		def: providers.Tool{
			Name:        c.name + "_update",
			Description: "Update a record in " + c.name + " by ID, replacing all of its fields. Fails and returns the current record if version is not the record's current version.",
			Parameters:  objectSchema(props, append([]string{"id", "version"}, names...)),
		},
		run: func(ctx context.Context, _ core.App, s Session, args json.RawMessage) (any, error) {
			var in versionArgs
			if err := decodeArgs(args, &in); err != nil {
				return nil, err
			}
			var v T
			if err := decodeArgs(args, &v); err != nil {
				return nil, err
			}
			return c.ifVersion(ctx, s, in, func(tx core.App, rec *core.Record) (any, error) {
				c.write(rec, v)
				if err := tx.SaveWithContext(ctx, rec); err != nil {
					return nil, err
				}
				return c.toolRecord(c.read(rec))
			})
		},
	}
}

func (c *Store[T]) deleteTool() tool {
	return tool{
		def: providers.Tool{
			Name:        c.name + "_delete",
			Description: "Delete a record in " + c.name + " by ID. Fails and returns the current record if version is not the record's current version.",
			Parameters:  objectSchema(map[string]any{"id": idProperty, "version": versionProperty}, []string{"id", "version"}),
		},
		run: func(ctx context.Context, _ core.App, s Session, args json.RawMessage) (any, error) {
			var in versionArgs
			if err := decodeArgs(args, &in); err != nil {
				return nil, err
			}
			return c.ifVersion(ctx, s, in, func(tx core.App, rec *core.Record) (any, error) {
				if err := tx.DeleteWithContext(ctx, rec); err != nil {
					return nil, err
				}
				return map[string]string{"deleted": in.ID}, nil
			})
		},
	}
}
