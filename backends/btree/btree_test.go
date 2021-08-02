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
	"math/rand"
	"testing"
	"time"

	"github.com/k3s-io/kine/pkg/server"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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

func TestBTreeCacheSimpleOperations(t *testing.T) {
	backend := NewBTreeCache(zap.NewExample())
	assert.Nil(t, backend.Start(context.Background()), "checking error")
	rev, kv, err := backend.Get(context.Background(), "/apisix/routes/123", 13)
	assert.Equal(t, rev, int64(1), "checking revision")
	assert.Nil(t, kv, "checking kv")
	assert.Nil(t, err, "checking error")

	rev, err = backend.Create(context.Background(), "/apisix/routes/123", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Nil(t, err, "checking error")

	// key already exists
	rev, err = backend.Create(context.Background(), "/apisix/routes/123", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Equal(t, server.ErrKeyExists, err, "checking error")

	// read it
	rev, kv, err = backend.Get(context.Background(), "/apisix/routes/123", 13)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/123",
		CreateRevision: 2,
		ModRevision:    2,
		Value:          []byte("{zxcvfda}"),
		Lease:          123,
	}, kv, "checking kv")
	assert.Nil(t, err, "checking error")

	// delete it.
	rev, kv, ok, err := backend.Delete(context.Background(), "/apisix/routes/123", 2)
	assert.Equal(t, int64(3), rev, "checking revision")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/123",
		CreateRevision: 2,
		ModRevision:    2,
		Value:          []byte("{zxcvfda}"),
		Lease:          123,
	}, kv, "checking kv")
	assert.Equal(t, true, ok, "checking success flag")
	assert.Nil(t, err, "checking error")
}

func TestBTreeCacheUpdate(t *testing.T) {
	backend := NewBTreeCache(zap.NewExample())
	rev, kv, ok, err := backend.Update(context.Background(), "/apisix/routes/123", []byte("{zxcvfda}"), 5, 123)
	assert.Equal(t, rev, int64(1), "checking revision")
	assert.Nil(t, kv, "checking kv")
	assert.Equal(t, false, ok, "checking success flag")
	assert.Nil(t, err, "checking error")

	// create it.
	rev, err = backend.Create(context.Background(), "/apisix/routes/123", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Nil(t, err, "checking error")

	// try to update it but failed.
	rev, kv, ok, err = backend.Update(context.Background(), "/apisix/routes/123", []byte("{zxcvfda}"), 5, 123)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/123",
		CreateRevision: 2,
		ModRevision:    2,
		Value:          []byte("{zxcvfda}"),
		Lease:          123,
	}, kv, "checking kv")
	assert.Equal(t, false, ok, "checking success flag")
	assert.Nil(t, err, "checking error")

	// try to update it but failed.
	rev, kv, ok, err = backend.Update(context.Background(), "/apisix/routes/123", []byte("{zxcvfda}"), 1, 123)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Nil(t, kv, "checking revision")
	assert.Equal(t, false, ok, "checking success flag")
	assert.Nil(t, err, "checking error")

	// update it
	rev, kv, ok, err = backend.Update(context.Background(), "/apisix/routes/123", []byte("{new value}"), 2, 123)
	assert.Equal(t, rev, int64(3), "checking revision")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/123",
		CreateRevision: 2,
		ModRevision:    3,
		Value:          []byte("{new value}"),
		Lease:          123,
	}, kv, "checking kv")
	assert.Equal(t, true, ok, "checking success flag")
	assert.Nil(t, err, "checking error")

	// read old version
	rev, kv, err = backend.Get(context.Background(), "/apisix/routes/123", 2)
	assert.Equal(t, rev, int64(3), "checking revision")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/123",
		CreateRevision: 2,
		ModRevision:    2,
		Value:          []byte("{zxcvfda}"),
		Lease:          123,
	}, kv, "checking kv")
	assert.Nil(t, err, "checking error")

	// read new version
	rev, kv, err = backend.Get(context.Background(), "/apisix/routes/123", 0)
	assert.Equal(t, rev, int64(3), "checking revision")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/123",
		CreateRevision: 2,
		ModRevision:    3,
		Value:          []byte("{new value}"),
		Lease:          123,
	}, kv, "checking kv")
	assert.Nil(t, err, "checking error")
}

func TestBTreeCacheDelete(t *testing.T) {
	backend := NewBTreeCache(zap.NewExample())
	rev, kv, ok, err := backend.Delete(context.Background(), "/apisix/routes/123", 5)
	assert.Equal(t, rev, int64(1), "checking revision")
	assert.Nil(t, kv, "checking kv")
	assert.Equal(t, false, ok, "checking success flag")
	assert.Nil(t, err, "checking error")

	// create it.
	rev, err = backend.Create(context.Background(), "/apisix/routes/123", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Nil(t, err, "checking error")

	// try to delete it but failed.
	rev, kv, ok, err = backend.Delete(context.Background(), "/apisix/routes/123", 3)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/123",
		CreateRevision: 2,
		ModRevision:    2,
		Value:          []byte("{zxcvfda}"),
		Lease:          123,
	}, kv, "checking kv")
	assert.Equal(t, false, ok, "checking success flag")
	assert.Nil(t, err, "checking error")

	// try to delete it but failed.
	rev, kv, ok, err = backend.Delete(context.Background(), "/apisix/routes/123", 1)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Nil(t, kv, "checking kv")
	assert.Equal(t, false, ok, "checking success flag")
	assert.Nil(t, err, "checking error")

	// try to delete it but failed.
	rev, kv, ok, err = backend.Delete(context.Background(), "/apisix/routes/123", 3)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/123",
		CreateRevision: 2,
		ModRevision:    2,
		Value:          []byte("{zxcvfda}"),
		Lease:          123,
	}, kv, "checking kv")
	assert.Equal(t, false, ok, "checking success flag")
	assert.Nil(t, err, "checking error")

	// delete it.
	rev, kv, ok, err = backend.Delete(context.Background(), "/apisix/routes/123", 2)
	assert.Equal(t, rev, int64(3), "checking revision")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/123",
		CreateRevision: 2,
		ModRevision:    2,
		Value:          []byte("{zxcvfda}"),
		Lease:          123,
	}, kv, "checking kv")
	assert.Equal(t, true, ok, "checking success flag")
	assert.Nil(t, err, "checking error")

	// read it.
	rev, kv, err = backend.Get(context.Background(), "/apisix/routes/123", 0)
	assert.Equal(t, rev, int64(3), "checking revision")
	assert.Nil(t, kv, "checking kv")
	assert.Nil(t, err, "checking error")
}

func TestBTreeCacheCount(t *testing.T) {
	backend := NewBTreeCache(zap.NewExample())

	rev, err := backend.Create(context.Background(), "/apisix/routes/123", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Nil(t, err, "checking error")

	rev, err = backend.Create(context.Background(), "/apisix/routes/134", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(3), "checking revision")
	assert.Nil(t, err, "checking error")

	rev, err = backend.Create(context.Background(), "/apisix/routes/1", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(4), "checking revision")
	assert.Nil(t, err, "checking error")

	rev, err = backend.Create(context.Background(), "/apisix/upstreams/1", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(5), "checking revision")
	assert.Nil(t, err, "checking error")

	rev, count, err := backend.Count(context.Background(), "/apisix")
	assert.Equal(t, int64(5), rev, "checking rev")
	assert.Equal(t, int64(4), count, "checking count")
	assert.Nil(t, err, "checking error")

	rev, count, err = backend.Count(context.Background(), "/apisix/routes")
	assert.Equal(t, int64(5), rev, "checking rev")
	assert.Equal(t, int64(3), count, "checking count")
	assert.Nil(t, err, "checking error")

	rev, count, err = backend.Count(context.Background(), "/apisix/upstreams")
	assert.Equal(t, int64(5), rev, "checking rev")
	assert.Equal(t, int64(1), count, "checking count")
	assert.Nil(t, err, "checking error")
}

func TestBTreeCacheList(t *testing.T) {
	backend := NewBTreeCache(zap.NewExample())

	rev, err := backend.Create(context.Background(), "/apisix/routes/123", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Nil(t, err, "checking error")

	rev, err = backend.Create(context.Background(), "/apisix/routes/134", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(3), "checking revision")
	assert.Nil(t, err, "checking error")

	rev, err = backend.Create(context.Background(), "/apisix/routes/1", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(4), "checking revision")
	assert.Nil(t, err, "checking error")

	rev, err = backend.Create(context.Background(), "/apisix/upstreams/1", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(5), "checking revision")
	assert.Nil(t, err, "checking error")

	rev, kvs, err := backend.List(context.Background(), "/apisix", "/apisix", 2, 1)
	assert.Equal(t, rev, int64(5), "checking revision")
	assert.Len(t, kvs, 0, "checking kvs")
	assert.Nil(t, err, "checking error")

	rev, kvs, err = backend.List(context.Background(), "/apisix", "/apisix", 2, 0)
	assert.Equal(t, rev, int64(5), "checking revision")
	assert.Len(t, kvs, 2, "checking kvs")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/1",
		CreateRevision: 4,
		ModRevision:    4,
		Value:          []byte("{zxcvfda}"),
		Lease:          123,
	}, kvs[0], "checking kv")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/123",
		CreateRevision: 2,
		ModRevision:    2,
		Value:          []byte("{zxcvfda}"),
		Lease:          123,
	}, kvs[1], "checking kv")
	assert.Nil(t, err, "checking error")

	rev, kvs, err = backend.List(context.Background(), "/apisix", "/apisix/routes/124", 2, 0)
	assert.Equal(t, rev, int64(5), "checking revision")
	assert.Len(t, kvs, 2, "checking kvs")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/routes/134",
		CreateRevision: 3,
		ModRevision:    3,
		Value:          []byte("{zxcvfda}"),
		Lease:          123,
	}, kvs[0], "checking kv")
	assert.Equal(t, &server.KeyValue{
		Key:            "/apisix/upstreams/1",
		CreateRevision: 5,
		ModRevision:    5,
		Value:          []byte("{zxcvfda}"),
		Lease:          123,
	}, kvs[1], "checking kv")
	assert.Nil(t, err, "checking error")
}

//func BenchmarkBTreeCacheGet(b *testing.B) {
//	cases := []struct {
//		name        string
//		prepareData func() []mystring
//	}{
//		{
//			name: "100 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//		{
//			name: "5000 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//		{
//			name: "20000 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//	}
//	for _, bc := range cases {
//		bc := bc
//		b.Run(bc.name, func(b *testing.B) {
//			data := bc.prepareData()
//			c := NewBTreeCache()
//			for _, item := range data {
//				c.Put(item)
//			}
//			b.ResetTimer()
//			for _, item := range data {
//				c.Get(item)
//			}
//		})
//	}
//}
//
//func BenchmarkBTreeCachePut(b *testing.B) {
//	cases := []struct {
//		name        string
//		prepareData func() []mystring
//	}{
//		{
//			name: "100 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//		{
//			name: "5000 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//		{
//			name: "20000 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//	}
//	for _, bc := range cases {
//		bc := bc
//		b.Run(bc.name, func(b *testing.B) {
//			data := bc.prepareData()
//			c := NewBTreeCache()
//			for _, item := range data {
//				c.Put(item)
//			}
//		})
//	}
//}
//
//func BenchmarkBTreeCacheRange(b *testing.B) {
//	cases := []struct {
//		name         string
//		prepareData  func() []mystring
//		prepareRange func(seed []mystring) ([]mystring, []mystring)
//	}{
//		{
//			name: "100 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//			prepareRange: func(seed []mystring) ([]mystring, []mystring) {
//				sort.Slice(seed, func(i, j int) bool {
//					return seed[i].Key() < seed[j].Key()
//				})
//				startPos := make([]mystring, 0, len(seed))
//				endPos := make([]mystring, 0, len(seed))
//				for i := 0; i < len(seed); i++ {
//					s := rand.Intn(len(seed))
//					e := rand.Intn(len(seed))
//					if s > e {
//						s, e = e, s
//					}
//					startPos = append(startPos, seed[s])
//					endPos = append(endPos, seed[e])
//				}
//				return startPos, endPos
//			},
//		},
//		{
//			name: "5000 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//			prepareRange: func(seed []mystring) ([]mystring, []mystring) {
//				sort.Slice(seed, func(i, j int) bool {
//					return seed[i].Key() < seed[j].Key()
//				})
//				startPos := make([]mystring, 0, len(seed))
//				endPos := make([]mystring, 0, len(seed))
//				for i := 0; i < len(seed); i++ {
//					s := rand.Intn(len(seed))
//					e := rand.Intn(len(seed))
//					if s > e {
//						s, e = e, s
//					}
//					startPos = append(startPos, seed[s])
//					endPos = append(endPos, seed[e])
//				}
//				return startPos, endPos
//			},
//		},
//		{
//			name: "20000 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//			prepareRange: func(seed []mystring) ([]mystring, []mystring) {
//				sort.Slice(seed, func(i, j int) bool {
//					return seed[i].Key() < seed[j].Key()
//				})
//				startPos := make([]mystring, 0, len(seed))
//				endPos := make([]mystring, 0, len(seed))
//				for i := 0; i < len(seed); i++ {
//					s := rand.Intn(len(seed))
//					e := rand.Intn(len(seed))
//					if s > e {
//						s, e = e, s
//					}
//					startPos = append(startPos, seed[s])
//					endPos = append(endPos, seed[e])
//				}
//				return startPos, endPos
//			},
//		},
//	}
//	for _, bc := range cases {
//		bc := bc
//		b.Run(bc.name, func(b *testing.B) {
//			data := bc.prepareData()
//			c := NewBTreeCache()
//			for _, item := range data {
//				c.Put(item)
//			}
//			startPos, endPos := bc.prepareRange(data)
//			b.ResetTimer()
//			for i := 0; i < len(data); i++ {
//				c.Range(startPos[i], endPos[i])
//			}
//		})
//	}
//}
//
//func BenchmarkBTreeCacheList(b *testing.B) {
//	cases := []struct {
//		name        string
//		prepareData func() []mystring
//	}{
//		{
//			name: "100 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//		{
//			name: "5000 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//		{
//			name: "20000 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//	}
//	for _, bc := range cases {
//		bc := bc
//		b.Run(bc.name, func(b *testing.B) {
//			data := bc.prepareData()
//			c := NewBTreeCache()
//			for _, item := range data {
//				c.Put(item)
//			}
//			b.ResetTimer()
//			for i := 0; i < 1000; i++ {
//				c.List()
//			}
//		})
//	}
//}
//
//func BenchmarkBTreeCacheDelete(b *testing.B) {
//	cases := []struct {
//		name        string
//		prepareData func() []mystring
//	}{
//		{
//			name: "100 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//		{
//			name: "5000 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//		{
//			name: "20000 items",
//			prepareData: func() []mystring {
//				var outputs []mystring
//				for i := 0; i < 100; i++ {
//					outputs = append(outputs, mystring(generateString(i*3+1)))
//				}
//				return outputs
//			},
//		},
//	}
//	for _, bc := range cases {
//		bc := bc
//		b.Run(bc.name, func(b *testing.B) {
//			data := bc.prepareData()
//			c := NewBTreeCache()
//			for _, item := range data {
//				c.Put(item)
//			}
//			b.ResetTimer()
//			for _, item := range data {
//				c.Delete(item)
//			}
//		})
//	}
//}
