package etcdserver

import (
	"context"
	"fmt"
	"github.com/api7/gopkg/pkg/log"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	mvccpb "go.etcd.io/etcd/api/v3/mvccpb"
	"go.uber.org/zap"
	"strings"
)

func (k *EtcdServer) Range(ctx context.Context, r *etcdserverpb.RangeRequest) (*etcdserverpb.RangeResponse, error) {
	var rev int64
	etcdKvs := []*mvccpb.KeyValue{}

	//if len(r.RangeEnd) > 0 {
	key := string(r.Key)
	levelOfKey := len(strings.Split(key, "/"))
	if levelOfKey == 3 {
		key = fmt.Sprintf("%s%%", key)
		log.Debug("TOP_LEVEL: ", key, " RANGE_END:", r.RangeEnd)
		revision, kvs, _ := k.backend.List(ctx, key, key, r.Limit, 0)

		for _, kv := range kvs {
			// If the last char is '/' ignore it ..
			lastChar := kv.Key[len(kv.Key)-1:]
			log.Debug("KEY:", kv.Key, " LAST:", lastChar)
			if lastChar == "/" {
				log.Debug("Last Char is '/'; skipping ...")
				continue
			}
			etcdKvs = append(etcdKvs, &mvccpb.KeyValue{
				Key:            []byte(kv.Key),
				Value:          kv.Value,
				ModRevision:    kv.ModRevision,
				CreateRevision: kv.CreateRevision,
				Lease:          kv.Lease,
			})
		}
		// DEBUG
		//spew.Dump(etcdKvs)
		rev = revision
	} else {
		log.Debug("KEY:", key, " LVL:", levelOfKey, " RANGE_END:", r.RangeEnd)
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
	_, kv, _ := k.backend.Get(ctx, key, "0", 0, 0)

	if kv != nil {
		revision, _, _, rerr := k.backend.Update(ctx, key, r.Value, kv.ModRevision, 0)
		log.Debugw("update", zap.String("key", key), zap.String("value", string(r.Value)), zap.Int64("revision", revision), zap.Error(rerr))
		err = rerr
		rev = revision
	} else {
		revision, rerr := k.backend.Create(ctx, key, r.Value, 0)
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
