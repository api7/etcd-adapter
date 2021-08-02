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

	"github.com/google/btree"
	"github.com/k3s-io/kine/pkg/server"
	"go.uber.org/zap"
)

const (
	// markedRevBytesLen is the byte length of marked revision.
	// The first `revBytesLen` bytes represents a normal revision. The last
	// one byte is the mark.
	markedRevBytesLen      = revBytesLen + 1
	markBytePosition       = markedRevBytesLen - 1
	markTombstone     byte = 't'
)

var (
	noPrefixEnd = []byte{0}
)

type btreeCache struct {
	sync.RWMutex
	currentRevision int64
	index           index
	logger          *zap.Logger
	tree            *btree.BTree
	watcherHub      map[string][]watcher
}

type watcher struct {
	ch chan []*server.Event
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

func (b *btreeCache) Get(ctx context.Context, key string, revision int64) (int64, *server.KeyValue, error) {
	b.RLock()
	defer b.RUnlock()
	return b.getLocked(ctx, key, revision)
}

func (b *btreeCache) getLocked(_ context.Context, key string, revision int64) (int64, *server.KeyValue, error) {
	if revision <= 0 {
		revision = b.currentRevision
	}

	modRev, createRev, _, err := b.index.Get([]byte(key), revision)
	if err != nil {
		if err == ErrRevisionNotFound {
			return b.currentRevision, nil, nil
		}
		return b.currentRevision, nil, err
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
	b.currentRevision++

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
	_, kv, err := b.getLocked(ctx, key, atRev)
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

func (b *btreeCache) List(_ context.Context, prefix, startKey string, limit, revision int64) (int64, []*server.KeyValue, error) {
	b.RLock()
	defer b.RUnlock()

	if revision <= 0 {
		revision = b.currentRevision
	}

	var (
		kvs   []*server.KeyValue
		count int64
	)

	end := getPrefixRangeEnd(prefix)
	keys, revs := b.index.Range([]byte(prefix), []byte(end), revision)
	for i, bkey := range keys {
		key := string(bkey)
		if key < startKey {
			continue
		}
		if limit > 0 && count >= limit {
			return b.currentRevision, kvs, nil
		}

		modRev, createRev, _, err := b.index.Get(bkey, revs[i].main)
		if err != nil {
			// Impossible to reach here
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
		kvs = append(kvs, &server.KeyValue{
			Key:            key,
			CreateRevision: createRev.main,
			ModRevision:    modRev.main,
			Value:          it.value,
			Lease:          it.lease,
		})
		count++
	}
	return b.currentRevision, kvs, nil
}

func (b *btreeCache) Delete(ctx context.Context, key string, atRev int64) (int64, *server.KeyValue, bool, error) {
	b.Lock()
	defer b.Unlock()
	_, kv, err := b.getLocked(ctx, key, atRev)
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

	end := getPrefixRangeEnd(prefix)
	keys, _ := b.index.Range([]byte(prefix), []byte(end), b.currentRevision)
	return b.currentRevision, int64(len(keys)), nil
}

func (b *btreeCache) Watch(ctx context.Context, key string, startRevision int64) <-chan []*server.Event {
	b.Lock()
	defer b.Unlock()
	w := watcher{
		ch: make(chan []*server.Event, 1), // prepare a slot so that the first send can be non-blocking.
	}
	if group, ok := b.watcherHub[key]; ok {
		group = append(group, w)
		b.watcherHub[key] = group
	} else {
		b.watcherHub[key] = []watcher{w}
	}

	revs := b.index.RangeSince([]byte(key), nil, startRevision)
	if len(revs) > 0 {
		var events []*server.Event
		for _, rev := range revs {
			_, kv, err := b.getLocked(ctx, key, rev.main)
			if err != nil {
				b.logger.Error("failed to query, ignore it",
					zap.String("key", key),
					zap.Int64("rev", rev.main),
				)
				continue
			}
			if kv != nil {
				events = append(events, &server.Event{
					Delete: false,
					Create: true,
					KV:     kv,
				})
			}
			if len(events) > 0 {
				w.ch <- events
			}
		}
	}
	return w.ch
}

func getPrefixRangeEnd(prefix string) string {
	return string(getPrefix([]byte(prefix)))
}

func getPrefix(key []byte) []byte {
	end := make([]byte, len(key))
	copy(end, key)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xff {
			end[i] = end[i] + 1
			end = end[:i+1]
			return end
		}
	}
	// next prefix does not exist (e.g., 0xffff);
	// default to WithFromKey policy
	return noPrefixEnd
}
