package ormtable

import (
	"context"

	"github.com/cosmos/cosmos-sdk/orm/model/ormlist"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/cosmos/cosmos-sdk/orm/encoding/ormkv"
	"github.com/cosmos/cosmos-sdk/orm/model/kvstore"
)

// Index defines an index on a table. Index instances
// are stateless, with all state existing only in the store passed
// to index methods.
type Index interface {

	// Iterator returns an iterator for this index with the provided list options.
	Iterator(ctx context.Context, options ...ormlist.Option) (Iterator, error)

	// MessageType returns the protobuf message type of the index.
	MessageType() protoreflect.MessageType

	// GetFieldNames returns the field names of the index.
	GetFieldNames() []protoreflect.Name

	// CompareKeys the two keys against the underlying IndexCodec, returning a
	// negative value if key1 is less than key2, 0 if they are equal, and a
	// positive value otherwise.
	CompareKeys(key1, key2 []protoreflect.Value) int

	// IsFullyOrdered returns true if all of the fields in the index are
	// considered "well-ordered" in terms of sorted iteration.
	IsFullyOrdered() bool

	doNotImplement()
}

// concreteIndex is used internally by table implementations.
type concreteIndex interface {
	Index
	ormkv.IndexCodec

	readValueFromIndexKey(context ReadBackend, primaryKey []protoreflect.Value, value []byte, message proto.Message) error
}

// UniqueIndex defines an unique index on a table.
type UniqueIndex interface {
	Index

	// Has returns true if the key values are present in the store for this index.
	Has(context context.Context, keyValues ...interface{}) (found bool, err error)

	// Get retrieves the message if one exists for the provided key values.
	Get(context context.Context, message proto.Message, keyValues ...interface{}) (found bool, err error)

	// DeleteByKey deletes the message if one exists in for the provided key values.
	DeleteByKey(context context.Context, keyValues ...interface{}) error
}

type indexer interface {
	onInsert(store kvstore.Writer, message protoreflect.Message) error
	onUpdate(store kvstore.Writer, new, existing protoreflect.Message) error
	onDelete(store kvstore.Writer, message protoreflect.Message) error
}