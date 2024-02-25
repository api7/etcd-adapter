package etcdserver

import (
	"context"
	"fmt"
	"github.com/api7/gopkg/pkg/log"
	"github.com/davecgh/go-spew/spew"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	mvccpb "go.etcd.io/etcd/api/v3/mvccpb"
	"go.uber.org/zap"
)

func (k *EtcdServer) Range(ctx context.Context, r *etcdserverpb.RangeRequest) (*etcdserverpb.RangeResponse, error) {
	var rev int64
	etcdKvs := []*mvccpb.KeyValue{}

	if len(r.RangeEnd) > 0 {

		revision, kvs, _ := k.backend.List(ctx, string(r.Key), string(r.Key), r.Limit, 0)

		for _, kv := range kvs {
			etcdKvs = append(etcdKvs, &mvccpb.KeyValue{
				Key:            []byte(kv.Key),
				Value:          kv.Value,
				ModRevision:    kv.ModRevision,
				CreateRevision: kv.CreateRevision,
				Lease:          kv.Lease,
			})
		}
		rev = revision
	} else {
		revision, kv, err := k.backend.Get(ctx, string(r.Key), "0", 0, 0)
		if err != nil {
			return nil, err
		}
		if kv != nil {
			etcdKvs = append(etcdKvs, &mvccpb.KeyValue{
				Key:            []byte(kv.Key),
				Value:          kv.Value,
				ModRevision:    kv.ModRevision,
				CreateRevision: kv.CreateRevision,
				Lease:          kv.Lease,
			})
		}
		rev = revision
	}

	return &etcdserverpb.RangeResponse{
		Header: &etcdserverpb.ResponseHeader{
			Revision: rev,
		},
		Kvs:   etcdKvs,
		Count: int64(len(etcdKvs)),
	}, nil
}

func (k *EtcdServer) Put(ctx context.Context, r *etcdserverpb.PutRequest) (*etcdserverpb.PutResponse, error) {
	key := string(r.Key)

	var (
		rev int64
		err error
	)
	id, kv, rerr := k.backend.Get(ctx, key, "0", 0, 0)

	if rerr != nil {
		log.Debugw("GET ERR:", zap.Error(rerr))
		return nil, rerr
	}

	log.Debugw("GET ID:", zap.Int64("id", id))
	spew.Dump(kv)

	if kv != nil {
		revision, kv, b, rerr := k.backend.Update(ctx, key, r.Value, kv.ModRevision, 0)
		log.Debugw("UPDATE:", zap.Int64("REV", revision), zap.Bool("OK?", b))
		spew.Dump(kv)
		if rerr != nil {
			log.Debugw("UPDATE ERR:", zap.Error(rerr))
			return nil, rerr
		}
		log.Debugw("update", zap.String("key", key), zap.String("value", string(r.Value)), zap.Int64("revision", revision), zap.Error(rerr))
		err = rerr
		rev = revision
	} else {
		revision, rerr := k.backend.Create(ctx, key, r.Value, 0)
		if rerr != nil {
			log.Debugw("CREATE ERR:", zap.Error(rerr))
			return nil, rerr
		}
		log.Debugw("CREATE:", zap.Int64("REV", revision))
		log.Debugw("create", zap.String("key", key), zap.String("value", string(r.Value)), zap.Int64("revision", revision), zap.Error(rerr))
		err = rerr
		rev = revision
	}
	return &etcdserverpb.PutResponse{
		Header: &etcdserverpb.ResponseHeader{
			Revision: rev,
		},
	}, err
}

// Only one deletion is supported, and range deletion is not supported.
// TODO: support delete range
func (k *EtcdServer) DeleteRange(ctx context.Context, r *etcdserverpb.DeleteRangeRequest) (*etcdserverpb.DeleteRangeResponse, error) {
	if r.RangeEnd != nil {
		return nil, fmt.Errorf("delete range is not supported")
	}
	_, prevKV, _ := k.backend.Get(ctx, string(r.Key), "0", 0, 0)
	rev, _, _, _ := k.backend.Delete(ctx, string(r.Key), prevKV.ModRevision)
	return &etcdserverpb.DeleteRangeResponse{
		Header: &etcdserverpb.ResponseHeader{
			Revision: rev,
		},
		Deleted: 1,
	}, nil
}
