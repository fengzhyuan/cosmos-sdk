package ormtable

import (
	"context"

	"github.com/cosmos/cosmos-sdk/orm/encoding/encodeutil"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/cosmos/cosmos-sdk/orm/encoding/ormkv"
)

// primaryKeyIndex defines an UniqueIndex for the primary key.
type primaryKeyIndex struct {
	*ormkv.PrimaryKeyCodec
	indexers       []indexer
	getBackend     func(context.Context) (Backend, error)
	getReadBackend func(context.Context) (ReadBackend, error)
}

func (p primaryKeyIndex) PrefixIterator(ctx context.Context, prefix []protoreflect.Value, options IteratorOptions) (Iterator, error) {
	backend, err := p.getReadBackend(ctx)
	if err != nil {
		return nil, err
	}

	prefixBz, err := p.EncodeKey(prefix)
	if err != nil {
		return nil, err
	}

	return prefixIterator(backend.CommitmentStoreReader(), backend, p, prefixBz, options)
}

func (p primaryKeyIndex) RangeIterator(ctx context.Context, start, end []protoreflect.Value, options IteratorOptions) (Iterator, error) {
	backend, err := p.getReadBackend(ctx)
	if err != nil {
		return nil, err
	}

	err = p.CheckValidRangeIterationKeys(start, end)
	if err != nil {
		return nil, err
	}

	startBz, err := p.EncodeKey(start)
	if err != nil {
		return nil, err
	}

	endBz, err := p.EncodeKey(end)
	if err != nil {
		return nil, err
	}

	fullEndKey := len(p.GetFieldNames()) == len(end)

	return rangeIterator(backend.CommitmentStoreReader(), backend, p, startBz, endBz, fullEndKey, options)
}

func (p primaryKeyIndex) doNotImplement() {}

func (p primaryKeyIndex) Has(context context.Context, key ...interface{}) (found bool, err error) {
	ctx, err := p.getReadBackend(context)
	if err != nil {
		return false, err
	}

	keyBz, err := p.EncodeKey(encodeutil.ValuesOf(key...))
	if err != nil {
		return false, err
	}

	return ctx.CommitmentStoreReader().Has(keyBz)
}

func (p primaryKeyIndex) Get(ctx context.Context, message proto.Message, values ...interface{}) (found bool, err error) {
	backend, err := p.getReadBackend(ctx)
	if err != nil {
		return false, err
	}

	return p.get(backend, message, encodeutil.ValuesOf(values...))
}

func (t primaryKeyIndex) DeleteByKey(ctx context.Context, primaryKeyValues ...interface{}) error {
	return t.doDeleteByKey(ctx, encodeutil.ValuesOf(primaryKeyValues...))
}

func (t primaryKeyIndex) doDeleteByKey(ctx context.Context, primaryKeyValues []protoreflect.Value) error {
	backend, err := t.getBackend(ctx)
	if err != nil {
		return err
	}

	pk, err := t.EncodeKey(primaryKeyValues)
	if err != nil {
		return err
	}

	msg := t.MessageType().New().Interface()
	found, err := t.getByKeyBytes(backend, pk, primaryKeyValues, msg)
	if err != nil {
		return err
	}

	if !found {
		return nil
	}

	if hooks := backend.getHooks(); hooks != nil {
		err = hooks.OnDelete(msg)
		if err != nil {
			return err
		}
	}

	// delete object
	writer := newBatchIndexCommitmentWriter(backend)
	defer writer.Close()
	err = writer.getCommitmentStore().Delete(pk)
	if err != nil {
		return err
	}

	// clear indexes
	mref := msg.ProtoReflect()
	indexStoreWriter := writer.getIndexStore()
	for _, idx := range t.indexers {
		err := idx.onDelete(indexStoreWriter, mref)
		if err != nil {
			return err
		}
	}

	return writer.Write()
}

func (p primaryKeyIndex) get(backend ReadBackend, message proto.Message, values []protoreflect.Value) (found bool, err error) {
	key, err := p.EncodeKey(values)
	if err != nil {
		return false, err
	}

	return p.getByKeyBytes(backend, key, values, message)
}

func (p primaryKeyIndex) getByKeyBytes(store ReadBackend, key []byte, keyValues []protoreflect.Value, message proto.Message) (found bool, err error) {
	bz, err := store.CommitmentStoreReader().Get(key)
	if err != nil {
		return false, err
	}

	if bz == nil {
		return false, nil
	}

	return true, p.Unmarshal(keyValues, bz, message)
}

func (p primaryKeyIndex) readValueFromIndexKey(_ ReadBackend, primaryKey []protoreflect.Value, value []byte, message proto.Message) error {
	return p.Unmarshal(primaryKey, value, message)
}

var _ UniqueIndex = &primaryKeyIndex{}