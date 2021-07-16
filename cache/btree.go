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

package cache

import (
	"sync"

	"github.com/google/btree"
)

type btreeCache struct {
	sync.RWMutex
	tree *btree.BTree
}

// item is the wrapper of user object so that we can implement the
// btree.Item interface.
type item struct {
	userOjbect Item
}

func (i *item) Less(j btree.Item) bool {
	return i.userOjbect.Less(j.(*item).userOjbect)
}

// NewBTreeCache returns a Cache interface which was implemented with
// the b-tree.
// Note this implementation is thread-safe. So feel free to use it among
// different goroutines.
func NewBTreeCache() Cache {
	return &btreeCache{
		tree: btree.New(32),
	}
}

func (b *btreeCache) Get(key Item) Item {
	b.RLock()
	defer b.RUnlock()

	// TODO: sync.Pool for item?
	v := b.tree.Get(&item{userOjbect: key})
	if v == nil {
		return nil
	}
	return v.(*item).userOjbect
}

func (b *btreeCache) Range(startKey, endKey Item) []Item {
	if startKey == nil || endKey == nil {
		panic("startKey or endKey is nil")
	}
	b.RLock()
	defer b.RUnlock()

	pivot := &item{
		userOjbect: startKey,
	}
	last := &item{
		userOjbect: endKey,
	}
	var items []Item
	b.tree.AscendGreaterOrEqual(pivot, func(curr btree.Item) bool {
		if !last.Less(curr) {
			items = append(items, curr.(*item).userOjbect)
			return true
		}
		return false
	})
	return items
}

func (b *btreeCache) Put(object Item) {
	oi := &item{
		userOjbect: object,
	}
	b.tree.ReplaceOrInsert(oi)
}

func (b *btreeCache) List() []Item {
	items := make([]Item, 0, b.tree.Len())
	b.tree.Ascend(func(i btree.Item) bool {
		items = append(items, i.(*item).userOjbect)
		return true
	})
	return items
}

func (b *btreeCache) Delete(object Item) {
	oi := &item{
		userOjbect: object,
	}
	b.tree.Delete(oi)
}
