// Copyright api7.ai
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package btree

import (
	"context"
	"sync"

	"go.etcd.io/etcd/clientv3"

	"github.com/k3s-io/kine/pkg/server"
	"go.uber.org/zap"

	"github.com/google/btree"
)

const (
	// markedRevBytesLen is the byte length of marked revision.
	// The first `revBytesLen` bytes represents a normal revision. The last
	// one byte is the mark.
	markedRevBytesLen      = revBytesLen + 1
	markBytePosition       = markedRevBytesLen - 1
	markTombstone     byte = 't'
)

type btreeCache struct {
	sync.RWMutex
	currentRevision int64
	index           index
	logger          *zap.Logger
	tree            *btree.BTree
}

// item is the wrapper of user object so that we can implement the
// backends.Item interface.
type item struct {
	key   revision
	value []byte
	lease int64
}

func (i *item) Less(j btree.Item) bool {
	left := i.key
	right := j.(*item).key
	return !(left == right || left.GreaterThan(right))
}

// NewBTreeCache returns a server.Backend interface which was implemented with
// the b-tree.
// Note this implementation is thread-safe. So feel free to use it among
// different goroutines.
func NewBTreeCache(logger *zap.Logger) server.Backend {
	return &btreeCache{
		currentRevision: 1,
		logger:          logger,
		tree:            btree.New(32),
		index:           newTreeIndex(logger),
	}
}

func (b *btreeCache) Start(_ context.Context) error {
	return nil
}

func (b *btreeCache) Get(_ context.Context, key string, revision int64) (int64, *server.KeyValue, error) {
	b.RLock()
	defer b.RUnlock()

	if revision <= 0 {
		revision = b.currentRevision
	}

	modRev, createRev, _, err := b.index.Get([]byte(key), revision)
	if err != nil {
		return 0, nil, err
	}

	// TODO: sync.Pool for item?
	v := b.tree.Get(&item{
		key: modRev,
	})
	if v == nil {
		return b.currentRevision, nil, nil
	}
	it := v.(*item)
	kv := &server.KeyValue{
		Key:            key,
		CreateRevision: createRev.main,
		ModRevision:    modRev.main,
		Value:          it.value,
		Lease:          it.lease,
	}
	return b.currentRevision, kv, nil
}

func (b *btreeCache) Create(_ context.Context, key string, value []byte, lease int64) (int64, error) {
	b.Lock()
	defer b.Unlock()
	if _, _, _, err := b.index.Get([]byte(key), b.currentRevision); err == nil || err != ErrRevisionNotFound {
		if err == nil {
			return b.currentRevision, server.ErrKeyExists
		}
		return b.currentRevision, err
	}

	rev := revision{
		main: b.currentRevision,
		sub:  0,
	}
	b.index.Put([]byte(key), rev)
	it := &item{
		key:   rev,
		value: value,
		lease: lease,
	}
	b.tree.ReplaceOrInsert(it)
	return rev.main, nil
}

func (b *btreeCache) Update(ctx context.Context, key string, value []byte, atRev, lease int64) (int64, *server.KeyValue, bool, error) {
	b.Lock()
	defer b.Unlock()
	_, kv, err := b.Get(ctx, key, atRev)
	if err != nil {
		return b.currentRevision, nil, false, err
	}
	if kv == nil {
		return b.currentRevision, nil, false, nil
	}
	if kv.ModRevision != atRev {
		return b.currentRevision, kv, false, nil
	}
	b.currentRevision++
	rev := revision{
		main: b.currentRevision,
	}
	b.index.Put([]byte(key), rev)
	it := &item{
		key:   rev,
		value: value,
		lease: lease,
	}
	b.tree.ReplaceOrInsert(it)
	return b.currentRevision, &server.KeyValue{
		Key:            key,
		Value:          value,
		CreateRevision: kv.CreateRevision,
		ModRevision:    b.currentRevision,
		Lease:          lease,
	}, true, nil
}

func (b *btreeCache) List(ctx context.Context, prefix, startKey string, limit, revision int64) (int64, []*server.KeyValue, error) {
	return 0, nil, nil
}

func (b *btreeCache) Delete(ctx context.Context, key string, atRev int64) (int64, *server.KeyValue, bool, error) {
	b.Lock()
	defer b.Unlock()
	_, kv, err := b.Get(ctx, key, atRev)
	if err != nil {
		return b.currentRevision, nil, false, err
	}
	if kv == nil {
		return b.currentRevision, nil, false, nil
	}
	if kv.ModRevision != atRev {
		return b.currentRevision, kv, false, nil
	}

	b.currentRevision++
	rev := revision{
		main: b.currentRevision,
	}
	if err := b.index.Tombstone([]byte(key), rev); err != nil {
		return b.currentRevision, nil, false, err
	}

	return b.currentRevision, kv, true, nil
}

func (b *btreeCache) Count(_ context.Context, prefix string) (int64, int64, error) {
	b.RLock()
	defer b.RUnlock()

	end := clientv3.GetPrefixRangeEnd(prefix)
	keys, _ := b.index.Range([]byte(prefix), []byte(end), b.currentRevision)
	return b.currentRevision, int64(len(keys)), nil
}

func (b *btreeCache) Watch(_ context.Context, key string, revision int64) <-chan []*server.Event {
	return nil
}
