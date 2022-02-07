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

func TestBTreeCacheWatch(t *testing.T) {
	backend := NewBTreeCache(zap.NewExample())
	assert.Nil(t, backend.Start(context.Background()))

	rev, err := backend.Create(context.Background(), "/apisix/routes/123", []byte("{zxcvfda}"), 123)
	assert.Equal(t, rev, int64(2), "checking revision")
	assert.Nil(t, err, "checking error")

	ctx, cancel := context.WithCancel(context.Background())
	ch := backend.Watch(ctx, "/apisix/routes", 0)
	evs := <-ch
	assert.Len(t, evs, 1, "checking the initial events")
	assert.Equal(t, evs[0].Create, true)
	assert.Equal(t, evs[0].Delete, false)
	assert.Nil(t, evs[0].PrevKV)
	assert.Equal(t, evs[0].KV.Key, "/apisix/routes/123")
	assert.Equal(t, evs[0].KV.Value, []byte("{zxcvfda}"))
	assert.Equal(t, evs[0].KV.CreateRevision, int64(2))
	assert.Equal(t, evs[0].KV.ModRevision, int64(2))

	lastRev := rev

	for _, v := range []string{"new value 1", "new value 2", "new value 3"} {
		rev, _, ok, err := backend.Update(context.Background(), "/apisix/routes/123", []byte(v), lastRev, 0)
		assert.Equal(t, ok, true, "checking update success flag")
		assert.Nil(t, err, "checking error")
		lastRev = rev
	}
	_, _, ok, err := backend.Delete(context.Background(), "/apisix/routes/123", lastRev)
	assert.Equal(t, ok, true, "checking delete success flag")
	assert.Nil(t, err, "checking error")

	_, err = backend.Create(context.Background(), "/apisix/routes/1333", []byte("{aaa}"), 3)
	assert.Nil(t, err, "checking error")

	evs = <-ch
	assert.Len(t, evs, 5)
	assert.Equal(t, evs[0].KV.Key, "/apisix/routes/123")
	assert.Equal(t, string(evs[0].KV.Value), "new value 1")
	assert.Equal(t, evs[0].Create, true)
	assert.Equal(t, evs[1].KV.Key, "/apisix/routes/123")
	assert.Equal(t, string(evs[1].KV.Value), "new value 2")
	assert.Equal(t, evs[1].Create, true)
	assert.Equal(t, evs[2].KV.Key, "/apisix/routes/123")
	assert.Equal(t, string(evs[2].KV.Value), "new value 3")
	assert.Equal(t, evs[2].Create, true)
	assert.Equal(t, evs[3].PrevKV.Key, "/apisix/routes/123")
	assert.Equal(t, string(evs[3].PrevKV.Value), "new value 3")
	assert.Equal(t, evs[3].Delete, true)
	assert.Equal(t, evs[4].KV.Key, "/apisix/routes/1333")
	assert.Equal(t, string(evs[4].KV.Value), "{aaa}")
	assert.Equal(t, evs[4].Create, true)

	// watcher will be removed.
	cancel()
	time.Sleep(time.Second)
	b := backend.(*btreeCache)
	assert.Len(t, b.watcherHub, 0)
}

func BenchmarkBTreeCacheGet(b *testing.B) {
	cases := []struct {
		name        string
		prepareData func() []string
	}{
		{
			name: "100 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
		{
			name: "5000 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
		{
			name: "20000 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
	}
	for _, bc := range cases {
		bc := bc
		b.Run(bc.name, func(b *testing.B) {
			data := bc.prepareData()
			c := NewBTreeCache(zap.NewExample())
			for _, item := range data {
				_, err := c.Create(context.Background(), item, []byte(item), 0)
				assert.Nil(b, err, "checking create error")
			}
			b.ResetTimer()
			for _, item := range data {
				_, kv, err := c.Get(context.Background(), item, 0)
				assert.Equal(b, kv.Key, item, "checking kv")
				assert.Nil(b, err, "checking get error")
			}
		})
	}
}

func BenchmarkBTreeCacheCreate(b *testing.B) {
	cases := []struct {
		name        string
		prepareData func() []string
	}{
		{
			name: "100 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
		{
			name: "5000 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
		{
			name: "20000 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
	}
	for _, bc := range cases {
		bc := bc
		b.Run(bc.name, func(b *testing.B) {
			data := bc.prepareData()
			c := NewBTreeCache(zap.NewExample())
			for _, item := range data {
				_, err := c.Create(context.Background(), item, []byte(item), 0)
				assert.Nil(b, err, "checking create error")
			}
		})
	}
}

func BenchmarkBTreeCacheList(b *testing.B) {
	cases := []struct {
		name        string
		prepareData func() []string
	}{
		{
			name: "100 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
		{
			name: "5000 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
		{
			name: "20000 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
	}
	for _, bc := range cases {
		bc := bc
		b.Run(bc.name, func(b *testing.B) {
			data := bc.prepareData()
			c := NewBTreeCache(zap.NewExample())
			for _, item := range data {
				_, err := c.Create(context.Background(), item, []byte(item), 0)
				assert.Nil(b, err, "checking create error")
			}
			b.ResetTimer()
			for i := 0; i < len(data); i++ {
				_, _, err := c.List(context.Background(), data[i], data[i], 0, 0)
				assert.Nil(b, err, "checking list error")
			}
		})
	}
}

func BenchmarkBTreeCacheDelete(b *testing.B) {
	cases := []struct {
		name        string
		prepareData func() []string
	}{
		{
			name: "100 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
		{
			name: "5000 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
		{
			name: "20000 items",
			prepareData: func() []string {
				var outputs []string
				for i := 0; i < 100; i++ {
					outputs = append(outputs, generateString(i*3+1))
				}
				return outputs
			},
		},
	}
	for _, bc := range cases {
		bc := bc
		b.Run(bc.name, func(b *testing.B) {
			data := bc.prepareData()
			c := NewBTreeCache(zap.NewExample())
			for _, item := range data {
				_, err := c.Create(context.Background(), item, []byte(item), 0)
				assert.Nil(b, err, "checking create error")
			}
			b.ResetTimer()
			for _, item := range data {
				_, _, _, err := c.Delete(context.Background(), item, 0)
				assert.Nil(b, err, "checking delete error")
			}
		})
	}
}
