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
	"container/list"
	"context"
	"strings"
	"sync"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/google/btree"
	"github.com/k3s-io/kine/pkg/server"
	"go.uber.org/zap"
)

var (
	noPrefixEnd = []byte{0}
)

type btreeCache struct {
	sync.RWMutex
	currentRevision int64
	index           index
	tree            *btree.BTree
	events          *list.List
	watcherHub      map[string]map[*watcher]struct{}
}

func (b *btreeCache) CurrentRevision(ctx context.Context) (int64, error) {
	return b.currentRevision, nil
}

func (b *btreeCache) Get(ctx context.Context, key, rangeEnd string, limit, revision int64) (int64, *server.KeyValue, error) {
	b.RLock()
	defer b.RUnlock()
	return b.getLocked(ctx, key, revision)
}

type watcher struct {
	startRev int64
	ch       chan []*server.Event
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
func NewBTreeCache() server.Backend {
	return &btreeCache{
		currentRevision: 1,
		tree:            btree.New(32),
		index:           newTreeIndex(),
		events:          list.New(),
		watcherHub:      make(map[string]map[*watcher]struct{}),
	}
}

func (b *btreeCache) Start(ctx context.Context) error {
	go b.sendEvents(ctx)
	return nil
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

func (b *btreeCache) DbSize(_ context.Context) (int64, error) {
	return 0, nil
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
	b.makeEvent(&server.KeyValue{
		Key:            key,
		CreateRevision: b.currentRevision,
		ModRevision:    b.currentRevision,
		Value:          value,
		Lease:          lease,
	}, nil, false)
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
	newKV := &server.KeyValue{
		Key:            key,
		Value:          value,
		CreateRevision: kv.CreateRevision,
		ModRevision:    b.currentRevision,
		Lease:          lease,
	}
	b.makeEvent(newKV, kv, false)
	return b.currentRevision, newKV, true, nil
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
	b.makeEvent(&server.KeyValue{
		Key:         key,
		ModRevision: b.currentRevision,
	}, kv, true)

	return b.currentRevision, kv, true, nil
}

func (b *btreeCache) Count(_ context.Context, prefix string) (int64, int64, error) {
	b.RLock()
	defer b.RUnlock()

	end := getPrefixRangeEnd(prefix)
	keys, _ := b.index.Range([]byte(prefix), []byte(end), b.currentRevision)
	return b.currentRevision, int64(len(keys)), nil
}
func (b *btreeCache) Watch(ctx context.Context, key string, startRevision int64) server.WatchResult {

	b.Lock()
	defer b.Unlock()
	w := &watcher{
		// use the current revision as the historical events will be handled at the first time.
		startRev: b.currentRevision + 1,
		ch:       make(chan []*server.Event, 1),
	}
	if group, ok := b.watcherHub[key]; ok {
		group[w] = struct{}{}
		b.watcherHub[key] = group
	} else {
		b.watcherHub[key] = map[*watcher]struct{}{
			w: {},
		}
	}
	go b.removeWatcher(ctx, key, w)

	pits := b.index.RangeSinceAll([]byte(key), []byte(getPrefixRangeEnd(key)), startRevision)
	if len(pits) > 0 {
		var events []*server.Event
		for _, pit := range pits {
			v := b.tree.Get(&item{
				key: pit.modifyRev,
			})
			if v == nil {
				// Should not happen.
				continue
			}
			it := v.(*item)
			kv := &server.KeyValue{
				Key:            string(pit.key),
				CreateRevision: pit.createRev.main,
				ModRevision:    pit.modifyRev.main,
				Value:          it.value,
				Lease:          it.lease,
			}
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

	return server.WatchResult{
		CurrentRevision: w.startRev,
		CompactRevision: w.startRev,
		Events:          w.ch,
	}
}

func (b *btreeCache) removeWatcher(ctx context.Context, key string, w *watcher) {
	<-ctx.Done()
	b.Lock()
	defer b.Unlock()

	group, ok := b.watcherHub[key]
	if !ok {
		// This shouldn't happen
		return
	}
	delete(group, w)
	if len(group) == 0 {
		delete(b.watcherHub, key)
	} else {
		b.watcherHub[key] = group
	}
	close(w.ch)
	log.Debug("removed a watcher",
		zap.String("key", key),
	)
}

// makeEvent makes an event and pushes it to the queue. Note this method should be
// invoked only if the mutex is locked.
func (b *btreeCache) makeEvent(kv, prevKV *server.KeyValue, deleteEvent bool) {
	var ev *server.Event
	if deleteEvent {
		ev = &server.Event{
			Delete: true,
			// Kine uses KV field to get the mod revision, so even for delete event,
			// add the kv field.
			KV:     kv,
			PrevKV: prevKV,
		}
	} else {
		ev = &server.Event{
			Create: true,
			KV:     kv,
			PrevKV: prevKV,
		}
	}
	b.events.PushBack(ev)
}

func (b *btreeCache) sendEvents(ctx context.Context) {
	// We clean up the event backlog per 500ms.
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			break
		}
		b.Lock()
		events := b.events
		b.events = list.New()
		aggregated := make(map[string][]*server.Event, len(b.watcherHub))
		for key := range b.watcherHub {
			aggregated[key] = []*server.Event{}
		}
		b.Unlock()

		go func(aggregated map[string][]*server.Event) {
			for {
				var (
					ev  *server.Event
					key string
				)
				if e := events.Front(); e != nil {
					ev = e.Value.(*server.Event)
					events.Remove(e)
				} else {
					break
				}

				if ev.KV != nil {
					key = ev.KV.Key
				} else {
					key = ev.PrevKV.Key
				}
				for k := range aggregated {
					// Prefix watch
					if strings.HasPrefix(key, k) {
						group := aggregated[k]
						aggregated[k] = append(group, ev)
					}
				}
			}
			b.RLock()
			for key, watchers := range b.watcherHub {
				events, ok := aggregated[key]
				if !ok || len(events) == 0 {
					continue
				}
				for w := range watchers {
					filtered := make([]*server.Event, 0, len(events))
					for _, ev := range events {
						var rev int64
						if ev.KV != nil {
							rev = ev.KV.ModRevision
						} else {
							rev = ev.PrevKV.ModRevision
						}
						if rev >= w.startRev {
							filtered = append(filtered, ev)
						}
					}
					if len(filtered) > 0 {
						go func(w *watcher) {
							// TODO we may deep-copy events if users want to modify them.
							w.ch <- filtered
						}(w)
					}
				}
			}
			b.RUnlock()
		}(aggregated)
	}
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
