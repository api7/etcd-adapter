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

	deepcopy bool
	tree *btree.BTree
}

// NewBTreeCache returns a Cache interface which was implemented with
// the b-tree. Note this implementation is thread-safe.
func NewBTreeCache(deepcopy bool) (Cache, error) {
	return &btreeCache{
		deepcopy: deepcopy,
		tree:     btree.New(32),
	}, nil
}

func (b *btreeCache) GetSingle(key string) (interface{}, error) {
}
