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

package etcdadapter

import (
	"context"
	"sync/atomic"

	"github.com/api7/etcd-adapter/cache"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a *adapter) Put(_ context.Context, r *etcdserverpb.PutRequest) (*etcdserverpb.PutResponse, error) {
	a.logger.Debug("put request arrived",
		zap.String("request", r.String()),
	)
	a.logger.Warn("put is not yet implemented ")

	return nil, status.New(codes.Unimplemented, "put is not yet implemented").Err()
}

func (a *adapter) Range(ctx context.Context, r *etcdserverpb.RangeRequest) (*etcdserverpb.RangeResponse, error) {
	a.logger.Debug("range request arrived",
		zap.String("request", r.String()),
	)

	var kvs []*mvccpb.KeyValue
	if r.RangeEnd == nil {
		raw := a.cache.Get(itemKey(r.Key))
		if raw != nil {
			kv, err := a.generateKeyValue(raw)
			if err != nil {
				return nil, err
			}
			kvs = []*mvccpb.KeyValue{kv}
		}
	} else {
		raw := a.cache.Range(itemKey(r.Key), itemKey(r.RangeEnd))
		if raw != nil {
			kvs = make([]*mvccpb.KeyValue, 0, len(raw))
			for _, rawi := range raw {
				kv, err := a.generateKeyValue(rawi)
				if err != nil {
					return nil, err
				}
				kvs = append(kvs, kv)
			}
		}
	}
	return &etcdserverpb.RangeResponse{
		Header: &etcdserverpb.ResponseHeader{
			Revision: atomic.LoadInt64(&a.revision),
		},
		Kvs:   kvs,
		Count: int64(len(kvs)),
	}, nil
}

func (a *adapter) DeleteRange(ctx context.Context, r *etcdserverpb.DeleteRangeRequest) (*etcdserverpb.DeleteRangeResponse, error) {
	a.logger.Debug("delete range request arrived",
		zap.String("request", r.String()),
	)

	a.logger.Warn("delete range is not yet implemented")
	return nil, rpctypes.ErrNotCapable
}

func (a *adapter) Txn(ctx context.Context, r *etcdserverpb.TxnRequest) (*etcdserverpb.TxnResponse, error) {
	a.logger.Debug("txn request arrived",
		zap.String("request", r.String()),
	)

	a.logger.Warn("txn is not yet implemented")
	return nil, rpctypes.ErrNotCapable
}

func (a *adapter) Compact(ctx context.Context, r *etcdserverpb.CompactionRequest) (*etcdserverpb.CompactionResponse, error) {
	a.logger.Debug("compact request arrived",
		zap.String("request", r.String()),
	)

	a.logger.Warn("compact is not yet implemented")

	return nil, rpctypes.ErrNotCapable
}

func (a *adapter) generateKeyValue(raw cache.Item) (*mvccpb.KeyValue, error) {
	ci := raw.(*cacheItem)
	value, err := ci.Item.Marshal()
	if err != nil {
		a.logger.Error("failed to marshal item",
			zap.Error(err),
			zap.Object("item", ci),
		)
		return nil, _errInternalError
	}

	return &mvccpb.KeyValue{
		Key:            []byte(ci.Item.Key()),
		CreateRevision: ci.createRevision,
		ModRevision:    ci.modRevision,
		Value:          value,
	}, nil
}
