package fdb

import (
	"context"
	"errors"
	"log"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/k3s-io/kine/pkg/server"
)

// Options 包含 FoundationDB 的配置信息
type Options struct {
	ClusterFile string
	Directory   []string
}

// fdbCache 是使用 FoundationDB 实现的后端缓存
type fdbCache struct {
	db  fdb.Database
	dir directory.DirectorySubspace
}

// NewFDBCache 返回一个实现了 server.Backend 接口的 FoundationDB 后端实例
func NewFDBCache(ctx context.Context, options *Options) (server.Backend, error) {
	fdb.MustAPIVersion(710)

	db, err := fdb.OpenDatabase(options.ClusterFile)
	if err != nil {
		return nil, err
	}

	dir, err := directory.CreateOrOpen(db, options.Directory, nil)
	if err != nil {
		return nil, err
	}

	fc := &fdbCache{
		db:  db,
		dir: dir,
	}
	return fc, nil
}

// Start 启动 FDB 后端
func (f *fdbCache) Start(ctx context.Context) error {
	log.Println("FoundationDB backend started")
	return nil
}

// Count 实现 server.Backend 接口的 Count 方法
func (f *fdbCache) Count(ctx context.Context, prefix string) (int64, int64, error) {
	count, err := f.db.ReadTransact(func(tr fdb.ReadTransaction) (interface{}, error) {
		rangeResult := tr.GetRange(f.dir.Sub(prefix), fdb.RangeOptions{}).GetSliceOrPanic()
		return int64(len(rangeResult)), nil
	})

	if err != nil {
		return 0, 0, err
	}

	revision := int64(1)
	return count.(int64), revision, nil
}

func (f *fdbCache) Get(ctx context.Context, key string, rangeEnd string, limit, revision int64) (int64, *server.KeyValue, error) {
	var kv *server.KeyValue
	var newRevision int64

	_, err := f.db.ReadTransact(func(tr fdb.ReadTransaction) (interface{}, error) {
		result := tr.Get(fdb.Key(key)).MustGet()
		log.Printf("Get: key=%s, result=%s", key, string(result))

		if len(result) != 0 {
			kv = &server.KeyValue{
				Key:   key,
				Value: result,
			}
			newRevision = revision
		}

		return nil, nil
	})

	if err != nil {
		log.Printf("Get: error=%v", err)
		return 0, nil, err
	}

	if kv == nil {
		log.Printf("Get: kv is nil for key=%s", key)
		return 0, nil, nil
	}

	log.Printf("Get: returning kv=%v, newRevision=%d", kv, newRevision)
	return newRevision, kv, nil
}

// Put 实现 server.Backend 接口的 Put 方法
func (f *fdbCache) Put(ctx context.Context, key string, value []byte) error {
	_, err := f.db.Transact(func(tr fdb.Transaction) (interface{}, error) {
		tr.Set(fdb.Key(key), value)
		return nil, nil
	})
	return err
}

// Delete 实现 server.Backend 接口的 Delete 方法
func (f *fdbCache) Delete(ctx context.Context, key string, revision int64) (int64, *server.KeyValue, bool, error) {
	var deletedRev int64
	var deletedKV *server.KeyValue
	var deleted bool

	_, err := f.db.Transact(func(tr fdb.Transaction) (interface{}, error) {
		val := tr.Get(fdb.Key(key)).MustGet()
		if len(val) == 0 {
			return nil, nil
		}

		tr.Clear(fdb.Key(key))

		deletedRev = revision
		deletedKV = &server.KeyValue{
			Key:   key,
			Value: val,
		}
		deleted = true

		return nil, nil
	})

	if err != nil {
		return 0, nil, false, err
	}

	return deletedRev, deletedKV, deleted, nil
}

// Create 实现 server.Backend 接口的 Create 方法
func (f *fdbCache) Create(ctx context.Context, key string, value []byte, revision int64) (int64, error) {
	rev, err := f.db.Transact(func(tr fdb.Transaction) (interface{}, error) {
		existing := tr.Get(fdb.Key(key)).MustGet()
		if len(existing) != 0 {
			return nil, errors.New("key already exists")
		}
		tr.Set(fdb.Key(key), value)
		return revision, nil
	})

	if err != nil {
		return 0, err
	}
	return rev.(int64), nil
}

// DbSize 实现 server.Backend 接口的 DbSize 方法
func (f *fdbCache) DbSize(ctx context.Context) (int64, error) {
	size, err := f.db.ReadTransact(func(tr fdb.ReadTransaction) (interface{}, error) {
		rangeResult := tr.GetRange(f.dir, fdb.RangeOptions{}).GetSliceOrPanic()
		var totalSize int64
		for _, kv := range rangeResult {
			totalSize += int64(len(kv.Key) + len(kv.Value))
		}
		return totalSize, nil
	})

	if err != nil {
		return 0, err
	}
	return size.(int64), nil
}

// List 实现 server.Backend 接口的 List 方法
func (f *fdbCache) List(ctx context.Context, prefix string, rangeEnd string, limit, revision int64) (int64, []*server.KeyValue, error) {
	var kvs []*server.KeyValue
	var newRevision int64

	_, err := f.db.ReadTransact(func(tr fdb.ReadTransaction) (interface{}, error) {
		rangeResult := tr.GetRange(f.dir.Sub(prefix), fdb.RangeOptions{Limit: int(limit)}).GetSliceOrPanic()

		for _, item := range rangeResult {
			if limit > 0 && int64(len(kvs)) >= limit {
				break
			}
			kv := &server.KeyValue{
				Key:   string(item.Key),
				Value: item.Value,
			}
			kvs = append(kvs, kv)
		}

		newRevision = revision
		return nil, nil
	})

	if err != nil {
		return 0, nil, err
	}

	return newRevision, kvs, nil
}

// Update 实现 server.Backend 接口的 Update 方法
func (f *fdbCache) Update(ctx context.Context, key string, value []byte, oldRevision, newRevision int64) (int64, *server.KeyValue, bool, error) {
	var updatedKV *server.KeyValue
	var updated bool

	_, err := f.db.Transact(func(tr fdb.Transaction) (interface{}, error) {
		existing := tr.Get(fdb.Key(key)).MustGet()

		if len(existing) == 0 {
			return nil, errors.New("key does not exist")
		}

		if oldRevision != newRevision {
			return nil, errors.New("revision mismatch")
		}

		tr.Set(fdb.Key(key), value)
		updatedKV = &server.KeyValue{
			Key:   key,
			Value: value,
		}
		updated = true

		return nil, nil
	})

	if err != nil {
		return 0, nil, false, err
	}
	return newRevision, updatedKV, updated, nil
}

// Watch 实现 server.Backend 接口的 Watch 方法
func (f *fdbCache) Watch(ctx context.Context, key string, revision int64) <-chan []*server.Event {

	events := make(chan []*server.Event)

	go func() {
		defer close(events)
	}()

	return events
}
