package kdnk

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/kdnk/providers"
)

// SingletonOption restricts which generated tools a singleton gets.
type SingletonOption interface {
	singletonOption()
}

type singletonRestriction int

func (singletonRestriction) singletonOption() {}

const (
	singletonReadOnly singletonRestriction = iota + 1
	singletonHidden
)

var (
	// SingletonReadOnly gives the model only the get tool of a per-user singleton. It does not
	// restrict Go code.
	SingletonReadOnly SingletonOption = singletonReadOnly
	// SingletonHidden gives the model no tools for the singleton. It does not restrict Go code.
	SingletonHidden SingletonOption = singletonHidden
)

const globalID = "global000000000"

// Value gives Go code access to a per-user singleton, one value per user.
type Value[T any] struct {
	s *singleton[T]
}

// GlobalValue gives Go code access to a global singleton, one value shared by every user.
type GlobalValue[T any] struct {
	s *singleton[T]
}

type singleton[T any] struct {
	store       Store[T]
	global      bool
	restriction singletonRestriction
}

// Singleton declares a per-user singleton named name whose fields come from T, and adds its
// generated tools to r. Each user has one value, stored as one row of a table named name. The last
// restriction in opts decides which tools are generated.
//
// T follows the rules of Collection, and the names id, owner, created and version are reserved.
// A user whose value was never set reads the zero T, and a field added to T reads as its zero
// value on rows stored before it existed. Rows are deleted with their user. The table is created
// and extended as for Collection, and its name must not clash with a collection or another
// singleton.
//
// The model gets name_get, which returns the user's value with a version derived from its fields,
// and name_set, which replaces all the fields. name_set takes the version the model last read, and
// changes nothing if the value has changed since; it then returns the current value instead.
func Singleton[T any](r *Registry, name string, opts ...SingletonOption) *Value[T] {
	return &Value[T]{newSingleton[T](r, name, false, opts)}
}

// Global declares a global singleton named name whose fields come from T, and adds its generated
// tools to r. There is one value, stored as one row of a table named name, and only Go code can
// set it.
//
// T follows the rules of Collection, and the names id, owner, created and version are reserved.
// The value reads as the zero T until it is first set, and a field added to T reads as its zero
// value until the value is next set. The table is created and extended as for Collection, and its
// name must not clash with a collection or another singleton.
//
// The model gets name_get, which returns the value with a version derived from its fields, unless
// opts include SingletonHidden. SingletonReadOnly has no effect.
func Global[T any](r *Registry, name string, opts ...SingletonOption) *GlobalValue[T] {
	return &GlobalValue[T]{newSingleton[T](r, name, true, opts)}
}

// Get returns s's user's value, or the zero T if it was never set. It returns an error if called
// before Run has started the app.
func (v *Value[T]) Get(ctx context.Context, s Session) (T, error) {
	return v.s.get(ctx, s.UserID)
}

// Set replaces all the fields of s's user's value. It returns an error if called before Run has
// started the app.
func (v *Value[T]) Set(ctx context.Context, s Session, val T) error {
	return v.s.set(ctx, s.UserID, val)
}

// Get returns the value, or the zero T if it was never set. It returns an error if called before
// Run has started the app.
func (g *GlobalValue[T]) Get(ctx context.Context) (T, error) {
	return g.s.get(ctx, "")
}

// Set replaces all the fields of the value. It returns an error if called before Run has started
// the app.
func (g *GlobalValue[T]) Set(ctx context.Context, val T) error {
	return g.s.set(ctx, "", val)
}

func newSingleton[T any](r *Registry, name string, global bool, opts []SingletonOption) *singleton[T] {
	s := &singleton[T]{store: Store[T]{name: name}, global: global}
	for _, o := range opts {
		if rs, ok := o.(singletonRestriction); ok {
			s.restriction = rs
		}
	}
	fields, err := structFields(reflect.TypeFor[T](), reservedFields)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("kdnk: singleton %q: %w", name, err))
	}
	s.store.fields = fields
	r.collections = append(r.collections, s)
	r.tools = append(r.tools, s.tools()...)
	return s
}

func (s *singleton[T]) collectionName() string { return s.store.name }

func (s *singleton[T]) bind(app core.App) { s.store.app = app }

func (s *singleton[T]) pbCollection() *core.Collection {
	col := s.store.pbCollection()
	col.Fields.RemoveByName("created")
	if s.global {
		col.Fields.RemoveByName("owner")
	} else {
		col.AddIndex("idx_"+s.store.name+"_owner", true, "owner", "")
	}
	return col
}

func (s *singleton[T]) find(ctx context.Context, app core.App, userID string) (*core.Record, error) {
	col, err := app.FindCachedCollectionByNameOrId(s.store.name)
	if err != nil {
		return nil, err
	}
	where := dbx.HashExp{"owner": userID}
	if s.global {
		where = dbx.HashExp{"id": globalID}
	}
	rec := &core.Record{}
	err = app.RecordQuery(col).WithContext(ctx).AndWhere(where).Limit(1).One(rec)
	if errors.Is(err, sql.ErrNoRows) {
		rec = core.NewRecord(col)
		if s.global {
			rec.Set("id", globalID)
		} else {
			rec.Set("owner", userID)
		}
		return rec, nil
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *singleton[T]) get(ctx context.Context, userID string) (T, error) {
	var zero T
	if s.store.app == nil {
		return zero, errUnbound
	}
	rec, err := s.find(ctx, s.store.app, userID)
	if err != nil {
		return zero, err
	}
	return s.store.read(rec).Data, nil
}

func (s *singleton[T]) set(ctx context.Context, userID string, v T) error {
	if s.store.app == nil {
		return errUnbound
	}
	rec, err := s.find(ctx, s.store.app, userID)
	if err != nil {
		return err
	}
	s.store.write(rec, v)
	return s.store.app.SaveWithContext(ctx, rec)
}

func (s *singleton[T]) tools() []tool {
	switch {
	case s.restriction == singletonHidden:
		return nil
	case s.global || s.restriction == singletonReadOnly:
		return []tool{s.getTool()}
	}
	return []tool{s.getTool(), s.setTool()}
}

func singletonResult[T any](v T) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	m := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m["version"], err = json.Marshal(versionOf(data)); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func (s *singleton[T]) getTool() tool {
	return tool{
		def: providers.Tool{
			Name:        s.store.name + "_get",
			Description: "Get " + s.store.name + ".",
			Parameters:  objectSchema(map[string]any{}, nil),
		},
		run: func(ctx context.Context, _ core.App, sess Session, _ json.RawMessage) (any, error) {
			v, err := s.get(ctx, sess.UserID)
			if err != nil {
				return nil, err
			}
			return singletonResult(v)
		},
	}
}

func (s *singleton[T]) setTool() tool {
	props, names := s.store.properties()
	props["version"] = versionProperty
	return tool{
		def: providers.Tool{
			Name:        s.store.name + "_set",
			Description: "Set " + s.store.name + ", replacing all of its fields. Fails and returns the current value if version is not the current version.",
			Parameters:  objectSchema(props, append([]string{"version"}, names...)),
		},
		run: func(ctx context.Context, _ core.App, sess Session, args json.RawMessage) (any, error) {
			var in versionArgs
			if err := decodeArgs(args, &in); err != nil {
				return nil, err
			}
			var v T
			if err := decodeArgs(args, &v); err != nil {
				return nil, err
			}
			if s.store.app == nil {
				return nil, errUnbound
			}
			var out any
			err := s.store.app.RunInTransaction(func(tx core.App) error {
				rec, err := s.find(ctx, tx, sess.UserID)
				if err != nil {
					return err
				}
				cur := s.store.read(rec).Data
				data, err := json.Marshal(cur)
				if err != nil {
					return err
				}
				if versionOf(data) != in.Version {
					b, err := singletonResult(cur)
					if err != nil {
						return err
					}
					out = map[string]any{"error": s.store.name + " was modified since it was read", "current": b}
					return nil
				}
				s.store.write(rec, v)
				if err := tx.SaveWithContext(ctx, rec); err != nil {
					return err
				}
				out, err = singletonResult(s.store.read(rec).Data)
				return err
			})
			return out, err
		},
	}
}
