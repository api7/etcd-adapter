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
	"testing"

	"github.com/stretchr/testify/assert"
)

type mystring string

func (s mystring) Less(t Item) bool {
	return s < t.(mystring)
}

func TestBTreeCacheGet(t *testing.T) {
	t.Parallel()
	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		c := NewBTreeCache()
		value := c.Get(mystring("abcdef"))
		assert.Nil(t, value, "lookup cache")
	})
	
	t.Run("lookup successfully", func(t *testing.T) {
		t.Parallel()
		c := NewBTreeCache()
		c.Put(mystring("a"))

		value := c.Get(mystring("a"))
		assert.Equal(t, "a", string(value.(mystring)), "checking the lookup result")
	})
}

func TestBTreeRange(t *testing.T) {
	t.Parallel()

	t.Run("sub-test 1", func(t *testing.T) {
		t.Parallel()
		c := NewBTreeCache()
		c.Put(mystring("a"))
		c.Put(mystring("b"))
		c.Put(mystring("d"))
		c.Put(mystring("e"))
		c.Put(mystring("aa"))
		c.Put(mystring("cc"))

		// most left
		items := c.Range(mystring("0"), mystring("a"))
		assert.Len(t, items, 1, "checking number of items")
		assert.Equal(t, mystring("a"), items[0].(mystring), "checking item")

		items = c.Range(mystring("b"), mystring("f"))
		assert.Len(t, items, 4, "checking number of items")
		assert.Equal(t, mystring("b"), items[0].(mystring), "checking item")
		assert.Equal(t, mystring("cc"), items[1].(mystring), "checking item")
		assert.Equal(t, mystring("d"), items[2].(mystring), "checking item")
		assert.Equal(t, mystring("e"), items[3].(mystring), "checking item")

		// most right
		items = c.Range(mystring("e"), mystring("xyz"))
		assert.Len(t, items, 1, "checking number of items")
		assert.Equal(t, mystring("e"), items[0].(mystring), "checking item")

		// none
		items = c.Range(mystring("fff"), mystring("xyz"))
		assert.Len(t, items, 0, "checking number of items")

		// startKey > endKey
		items = c.Range(mystring("b"), mystring("a"))
		assert.Len(t, items, 0, "checking number of items")
	})
}

func TestBTreeCacheList(t *testing.T) {
	t.Parallel()

	t.Run("sub-test 1", func(t *testing.T) {
		t.Parallel()
		c := NewBTreeCache()
		c.Put(mystring("a"))
		c.Put(mystring("b"))
		c.Put(mystring("d"))
		c.Put(mystring("e"))
		c.Put(mystring("aa"))
		c.Put(mystring("cc"))

		items := c.List()
		assert.Len(t, items, 6, "checking number of items")
		assert.Equal(t, mystring("a"), items[0].(mystring), "checking item")
		assert.Equal(t, mystring("aa"), items[1].(mystring), "checking item")
		assert.Equal(t, mystring("b"), items[2].(mystring), "checking item")
		assert.Equal(t, mystring("cc"), items[3].(mystring), "checking item")
		assert.Equal(t, mystring("d"), items[4].(mystring), "checking item")
		assert.Equal(t, mystring("e"), items[5].(mystring), "checking item")
	})

}
