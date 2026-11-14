package agentsmith

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a record does not exist or belongs to another user.
var ErrNotFound = errors.New("agentsmith: not found")

// CollectionOption restricts which generated tools a collection gets.
type CollectionOption interface {
	collectionOption()
}

type restriction int

func (restriction) collectionOption() {}

const (
	appendOnly restriction = iota + 1
	readOnly
)

var (
	// AppendOnly limits the model to creating, listing and getting records. It does not restrict
	// Go code.
	AppendOnly CollectionOption = appendOnly
	// ReadOnly limits the model to listing and getting records. It does not restrict Go code.
	ReadOnly CollectionOption = readOnly
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
}

// Collection declares a collection named name whose fields come from T, and adds its generated
// tools to r. The last restriction in opts decides which tools are generated. The collection
// itself is not recorded on r yet, and the generated tools always return an error.
func Collection[T any](r *Registry, name string, opts ...CollectionOption) *Store[T] {
	c := &Store[T]{name: name}
	for _, o := range opts {
		if rs, ok := o.(restriction); ok {
			c.restriction = rs
		}
	}
	r.tools = append(r.tools, c.tools()...)
	return c
}

// Create adds a record owned by s's user. It is not implemented yet and always returns an error.
func (c *Store[T]) Create(ctx context.Context, s Session, v T) (Record[T], error) {
	return Record[T]{}, errNotImplemented
}

// List returns up to limit of s's user's records. It is not implemented yet and always returns
// an error.
func (c *Store[T]) List(ctx context.Context, s Session, limit int) ([]Record[T], error) {
	return nil, errNotImplemented
}

// Get returns the record with the given id if s's user owns it. It is not implemented yet and
// always returns an error.
func (c *Store[T]) Get(ctx context.Context, s Session, id string) (Record[T], error) {
	return Record[T]{}, errNotImplemented
}

// Update replaces the fields of the record with the given id if s's user owns it. It is not
// implemented yet and always returns an error.
func (c *Store[T]) Update(ctx context.Context, s Session, id string, v T) (Record[T], error) {
	return Record[T]{}, errNotImplemented
}

// Delete removes the record with the given id if s's user owns it. It is not implemented yet and
// always returns an error.
func (c *Store[T]) Delete(ctx context.Context, s Session, id string) error {
	return errNotImplemented
}
