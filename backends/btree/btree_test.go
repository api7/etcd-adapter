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
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var _letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = _letterRunes[rand.Intn(len(_letterRunes))]
	}
	return string(b)
}

type mystring string

func (s mystring) Key() string {
	return string(s)
}

func (s mystring) Marshal() ([]byte, error) {
	return []byte(s), nil
}

func TestBTreeCacheGet(t *testing.T) {
	t.Parallel()
	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		c := NewBTreeCache()
		value := c.Get(mystring("abcdef"))
		assert.Nil(t, value, "lookup backends")
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
		items := c.Range(mystring("0"), mystring("aa"))
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

func TestBTreeCacheDelete(t *testing.T) {
	t.Parallel()
	t.Run("sub-test 1", func(t *testing.T) {
		t.Parallel()
		c := NewBTreeCache()
		c.Put(mystring("aa"))
		item := c.Get(mystring("aa")).(mystring)
		assert.Equal(t, mystring("aa"), item, "checking item")
		c.Delete(mystring("aa"))
		assert.Nil(t, c.Get(mystring("aa")), "checking the item after deleting it")
	})
}

func BenchmarkBTreeCacheGet(b *testing.B) {
	cases := []struct {
		name        string
		prepareData func() []mystring
	}{
		{
			name: "100 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
		{
			name: "5000 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
		{
			name: "20000 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
	}
	for _, bc := range cases {
		bc := bc
		b.Run(bc.name, func(b *testing.B) {
			data := bc.prepareData()
			c := NewBTreeCache()
			for _, item := range data {
				c.Put(item)
			}
			b.ResetTimer()
			for _, item := range data {
				c.Get(item)
			}
		})
	}
}

func BenchmarkBTreeCachePut(b *testing.B) {
	cases := []struct {
		name        string
		prepareData func() []mystring
	}{
		{
			name: "100 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
		{
			name: "5000 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
		{
			name: "20000 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
	}
	for _, bc := range cases {
		bc := bc
		b.Run(bc.name, func(b *testing.B) {
			data := bc.prepareData()
			c := NewBTreeCache()
			for _, item := range data {
				c.Put(item)
			}
		})
	}
}

func BenchmarkBTreeCacheRange(b *testing.B) {
	cases := []struct {
		name         string
		prepareData  func() []mystring
		prepareRange func(seed []mystring) ([]mystring, []mystring)
	}{
		{
			name: "100 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
			prepareRange: func(seed []mystring) ([]mystring, []mystring) {
				sort.Slice(seed, func(i, j int) bool {
					return seed[i].Key() < seed[j].Key()
				})
				startPos := make([]mystring, 0, len(seed))
				endPos := make([]mystring, 0, len(seed))
				for i := 0; i < len(seed); i++ {
					s := rand.Intn(len(seed))
					e := rand.Intn(len(seed))
					if s > e {
						s, e = e, s
					}
					startPos = append(startPos, seed[s])
					endPos = append(endPos, seed[e])
				}
				return startPos, endPos
			},
		},
		{
			name: "5000 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
			prepareRange: func(seed []mystring) ([]mystring, []mystring) {
				sort.Slice(seed, func(i, j int) bool {
					return seed[i].Key() < seed[j].Key()
				})
				startPos := make([]mystring, 0, len(seed))
				endPos := make([]mystring, 0, len(seed))
				for i := 0; i < len(seed); i++ {
					s := rand.Intn(len(seed))
					e := rand.Intn(len(seed))
					if s > e {
						s, e = e, s
					}
					startPos = append(startPos, seed[s])
					endPos = append(endPos, seed[e])
				}
				return startPos, endPos
			},
		},
		{
			name: "20000 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
			prepareRange: func(seed []mystring) ([]mystring, []mystring) {
				sort.Slice(seed, func(i, j int) bool {
					return seed[i].Key() < seed[j].Key()
				})
				startPos := make([]mystring, 0, len(seed))
				endPos := make([]mystring, 0, len(seed))
				for i := 0; i < len(seed); i++ {
					s := rand.Intn(len(seed))
					e := rand.Intn(len(seed))
					if s > e {
						s, e = e, s
					}
					startPos = append(startPos, seed[s])
					endPos = append(endPos, seed[e])
				}
				return startPos, endPos
			},
		},
	}
	for _, bc := range cases {
		bc := bc
		b.Run(bc.name, func(b *testing.B) {
			data := bc.prepareData()
			c := NewBTreeCache()
			for _, item := range data {
				c.Put(item)
			}
			startPos, endPos := bc.prepareRange(data)
			b.ResetTimer()
			for i := 0; i < len(data); i++ {
				c.Range(startPos[i], endPos[i])
			}
		})
	}
}

func BenchmarkBTreeCacheList(b *testing.B) {
	cases := []struct {
		name        string
		prepareData func() []mystring
	}{
		{
			name: "100 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
		{
			name: "5000 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
		{
			name: "20000 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
	}
	for _, bc := range cases {
		bc := bc
		b.Run(bc.name, func(b *testing.B) {
			data := bc.prepareData()
			c := NewBTreeCache()
			for _, item := range data {
				c.Put(item)
			}
			b.ResetTimer()
			for i := 0; i < 1000; i++ {
				c.List()
			}
		})
	}
}

func BenchmarkBTreeCacheDelete(b *testing.B) {
	cases := []struct {
		name        string
		prepareData func() []mystring
	}{
		{
			name: "100 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
		{
			name: "5000 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
		{
			name: "20000 items",
			prepareData: func() []mystring {
				var outputs []mystring
				for i := 0; i < 100; i++ {
					outputs = append(outputs, mystring(generateString(i*3+1)))
				}
				return outputs
			},
		},
	}
	for _, bc := range cases {
		bc := bc
		b.Run(bc.name, func(b *testing.B) {
			data := bc.prepareData()
			c := NewBTreeCache()
			for _, item := range data {
				c.Put(item)
			}
			b.ResetTimer()
			for _, item := range data {
				c.Delete(item)
			}
		})
	}
}
