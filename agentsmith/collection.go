package agentsmith

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ErrNotFound is returned when a record does not exist or belongs to another user.
var ErrNotFound = errors.New("agentsmith: not found")

var errUnbound = errors.New("agentsmith: collection used before Run started the app")

// CollectionOption restricts which generated tools a collection gets.
type CollectionOption interface {
	collectionOption()
}

type restriction int

func (restriction) collectionOption() {}

const (
	appendOnly restriction = iota + 1
	readOnly
	hidden
)

var (
	// AppendOnly limits the model to creating, listing and getting records. It does not restrict
	// Go code.
	AppendOnly CollectionOption = appendOnly
	// ReadOnly limits the model to listing and getting records. It does not restrict Go code.
	ReadOnly CollectionOption = readOnly
	// Hidden gives the model no tools for the collection. It does not restrict Go code.
	Hidden CollectionOption = hidden
)

// Record is a stored record of a collection whose fields are T.
type Record[T any] struct {
	// ID is the record's ID.
	ID string
	// Created is when the record was created.
	Created time.Time
	// Data holds the record's fields.
	Data T
}

// Store gives Go code access to a collection's records, scoped to a session's user.
type Store[T any] struct {
	name        string
	restriction restriction
	fields      []field
	app         core.App
}

type collection interface {
	collectionName() string
	pbCollection() *core.Collection
	bind(app core.App)
}

type fieldKind int

const (
	kindText fieldKind = iota
	kindBool
	kindInt
	kindUint
	kindFloat
	kindDate
)

type field struct {
	index     []int
	name      string
	desc      string
	kind      fieldKind
	omitEmpty bool
}

var timeType = reflect.TypeFor[time.Time]()

var reservedFields = []string{"id", "owner", "created"}

// Collection declares a collection named name whose fields come from T, and adds its generated
// tools to r. The last restriction in opts decides which tools are generated.
//
// T must be a struct. Each exported field becomes a column named by its json tag, or by the Go
// field name if the tag has no name; fields tagged json:"-" are skipped. A jsonschema tag gives
// the field's description in the generated tools. Supported field types are strings, bools,
// integers, floats and time.Time, including named types with those underlying kinds. Embedded
// fields, other types, the names id, owner and created, and two fields with the same name make
// Run return an error.
//
// Every record also has a hidden owner, the user who created it, and a created time. When the
// server starts serving, the table is created if it does not exist; if it does, missing columns
// are added, and a column whose type differs makes serving fail. Columns are never changed or
// removed. Records are deleted with their owner.
func Collection[T any](r *Registry, name string, opts ...CollectionOption) *Store[T] {
	c := &Store[T]{name: name}
	for _, o := range opts {
		if rs, ok := o.(restriction); ok {
			c.restriction = rs
		}
	}
	fields, err := structFields(reflect.TypeFor[T](), reservedFields)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("agentsmith: collection %q: %w", name, err))
	}
	c.fields = fields
	r.collections = append(r.collections, c)
	r.tools = append(r.tools, c.tools()...)
	return c
}

func structFields(t reflect.Type, reserved []string) ([]field, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("type %s is not a struct", t)
	}
	var fields []field
	seen := map[string]bool{}
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.Anonymous {
			return nil, fmt.Errorf("embedded field %s is not supported", sf.Name)
		}
		if !sf.IsExported() {
			continue
		}
		tag := sf.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if name == "" {
			name = sf.Name
		}
		for _, r := range reserved {
			if strings.EqualFold(name, r) {
				return nil, fmt.Errorf("field name %q is reserved", name)
			}
		}
		if seen[strings.ToLower(name)] {
			return nil, fmt.Errorf("field name %q is used twice", name)
		}
		seen[strings.ToLower(name)] = true
		kind, ok := kindOf(sf.Type)
		if !ok {
			return nil, fmt.Errorf("field %s has unsupported type %s", sf.Name, sf.Type)
		}
		omitEmpty := false
		for _, o := range strings.Split(opts, ",") {
			omitEmpty = omitEmpty || o == "omitempty"
		}
		fields = append(fields, field{index: sf.Index, name: name, desc: sf.Tag.Get("jsonschema"), kind: kind, omitEmpty: omitEmpty})
	}
	return fields, nil
}

func kindOf(t reflect.Type) (fieldKind, bool) {
	if t == timeType {
		return kindDate, true
	}
	switch t.Kind() {
	case reflect.String:
		return kindText, true
	case reflect.Bool:
		return kindBool, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return kindInt, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return kindUint, true
	case reflect.Float32, reflect.Float64:
		return kindFloat, true
	}
	return 0, false
}

func (c *Store[T]) collectionName() string { return c.name }

func (c *Store[T]) bind(app core.App) { c.app = app }

func (c *Store[T]) pbCollection() *core.Collection {
	col := core.NewBaseCollection(c.name)
	col.Fields.Add(&core.RelationField{Name: "owner", CollectionId: "users", Required: true, MaxSelect: 1, CascadeDelete: true})
	for _, f := range c.fields {
		switch f.kind {
		case kindText:
			col.Fields.Add(&core.TextField{Name: f.name})
		case kindBool:
			col.Fields.Add(&core.BoolField{Name: f.name})
		case kindInt, kindUint:
			col.Fields.Add(&core.NumberField{Name: f.name, OnlyInt: true})
		case kindFloat:
			col.Fields.Add(&core.NumberField{Name: f.name})
		case kindDate:
			col.Fields.Add(&core.DateField{Name: f.name})
		}
	}
	col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	return col
}

func (c *Store[T]) collection() (*core.Collection, error) {
	if c.app == nil {
		return nil, errUnbound
	}
	return c.app.FindCachedCollectionByNameOrId(c.name)
}

func (c *Store[T]) write(rec *core.Record, v T) {
	rv := reflect.ValueOf(v)
	for _, f := range c.fields {
		fv := rv.FieldByIndex(f.index)
		switch f.kind {
		case kindText:
			rec.Set(f.name, fv.String())
		case kindBool:
			rec.Set(f.name, fv.Bool())
		case kindInt:
			rec.Set(f.name, fv.Int())
		case kindUint:
			rec.Set(f.name, fv.Uint())
		case kindFloat:
			rec.Set(f.name, fv.Float())
		case kindDate:
			rec.Set(f.name, fv.Interface().(time.Time))
		}
	}
}

func (c *Store[T]) read(rec *core.Record) Record[T] {
	var v T
	rv := reflect.ValueOf(&v).Elem()
	for _, f := range c.fields {
		fv := rv.FieldByIndex(f.index)
		switch f.kind {
		case kindText:
			fv.SetString(rec.GetString(f.name))
		case kindBool:
			fv.SetBool(rec.GetBool(f.name))
		case kindInt:
			fv.SetInt(int64(rec.GetFloat(f.name)))
		case kindUint:
			fv.SetUint(uint64(rec.GetFloat(f.name)))
		case kindFloat:
			fv.SetFloat(rec.GetFloat(f.name))
		case kindDate:
			fv.Set(reflect.ValueOf(storedTime(rec, f.name)))
		}
	}
	return Record[T]{ID: rec.Id, Created: storedTime(rec, "created"), Data: v}
}

func storedTime(rec *core.Record, name string) time.Time {
	return rec.GetDateTime(name).Time().UTC().Truncate(time.Millisecond)
}

func (c *Store[T]) find(ctx context.Context, s Session, id string) (*core.Record, error) {
	col, err := c.collection()
	if err != nil {
		return nil, err
	}
	rec := &core.Record{}
	err = c.app.RecordQuery(col).WithContext(ctx).
		AndWhere(dbx.HashExp{"id": id, "owner": s.UserID}).
		Limit(1).
		One(rec)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// Create adds a record owned by s's user and returns it. It returns an error if called before
// Run has started the app.
func (c *Store[T]) Create(ctx context.Context, s Session, v T) (Record[T], error) {
	col, err := c.collection()
	if err != nil {
		return Record[T]{}, err
	}
	rec := core.NewRecord(col)
	rec.Set("owner", s.UserID)
	c.write(rec, v)
	if err := c.app.SaveWithContext(ctx, rec); err != nil {
		return Record[T]{}, err
	}
	return c.read(rec), nil
}

// List returns up to limit of s's user's records, newest first. It returns an error if limit is
// not positive, or if called before Run has started the app.
func (c *Store[T]) List(ctx context.Context, s Session, limit int) ([]Record[T], error) {
	if limit <= 0 {
		return nil, errors.New("agentsmith: limit must be positive")
	}
	col, err := c.collection()
	if err != nil {
		return nil, err
	}
	var recs []*core.Record
	err = c.app.RecordQuery(col).WithContext(ctx).
		AndWhere(dbx.HashExp{"owner": s.UserID}).
		OrderBy("created DESC").
		Limit(int64(limit)).
		All(&recs)
	if err != nil {
		return nil, err
	}
	out := make([]Record[T], 0, len(recs))
	for _, rec := range recs {
		out = append(out, c.read(rec))
	}
	return out, nil
}

// Get returns the record with the given id if s's user owns it, and ErrNotFound otherwise. It
// returns an error if called before Run has started the app.
func (c *Store[T]) Get(ctx context.Context, s Session, id string) (Record[T], error) {
	rec, err := c.find(ctx, s, id)
	if err != nil {
		return Record[T]{}, err
	}
	return c.read(rec), nil
}

// Update replaces all the fields of the record with the given id if s's user owns it, and
// returns the updated record. It returns ErrNotFound if s's user does not own such a record, and
// an error if called before Run has started the app.
func (c *Store[T]) Update(ctx context.Context, s Session, id string, v T) (Record[T], error) {
	rec, err := c.find(ctx, s, id)
	if err != nil {
		return Record[T]{}, err
	}
	c.write(rec, v)
	if err := c.app.SaveWithContext(ctx, rec); err != nil {
		return Record[T]{}, err
	}
	return c.read(rec), nil
}

// Delete removes the record with the given id if s's user owns it. It returns ErrNotFound if s's
// user does not own such a record, and an error if called before Run has started the app.
func (c *Store[T]) Delete(ctx context.Context, s Session, id string) error {
	rec, err := c.find(ctx, s, id)
	if err != nil {
		return err
	}
	return c.app.DeleteWithContext(ctx, rec)
}
