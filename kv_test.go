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
	"testing"

	"github.com/api7/etcd-adapter/cache"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.uber.org/zap"
)

func TestPutRequest(t *testing.T) {
	t.Parallel()
	adapter := &adapter{
		revision: 0,
		logger:   zap.NewExample(),
	}
	resp, err := adapter.Put(context.Background(), nil)
	assert.Nil(t, resp, "checking put response")
	assert.Equal(t, "rpc error: code = Unimplemented desc = put is not yet implemented", err.Error())
}

func TestRangeRequest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		rr        *etcdserverpb.RangeRequest
		resp      *etcdserverpb.RangeResponse
		prepareFn func() *adapter
	}{
		{
			name: "exact query",
			rr: &etcdserverpb.RangeRequest{
				Key: []byte("Alice"),
			},
			prepareFn: func() *adapter {
				a := &adapter{
					revision: 13,
					logger:   zap.NewExample(),
					cache:    cache.NewBTreeCache(),
				}
				items := []*cacheItem{
					{
						Item: itemKey("Alex"),
					},
					{
						Item: itemKey("Alice"),
					},
					{
						Item: itemKey("Bob"),
					},
				}
				a.cache.Put(items[0])
				a.cache.Put(items[1])
				a.cache.Put(items[2])
				return a
			},
			resp: &etcdserverpb.RangeResponse{
				Header: &etcdserverpb.ResponseHeader{
					Revision: 13,
				},
				Kvs: []*mvccpb.KeyValue{
					{
						Key:   []byte("Alice"),
						Value: []byte("Alice"),
					},
				},
			},
		},
		{
			name: "range query",
			rr: &etcdserverpb.RangeRequest{
				Key:      []byte("Alice"),
				RangeEnd: []byte("Cather"),
			},
			prepareFn: func() *adapter {
				a := &adapter{
					revision: 13,
					logger:   zap.NewExample(),
					cache:    cache.NewBTreeCache(),
				}
				items := []*cacheItem{
					{
						Item: itemKey("Alex"),
					},
					{
						Item: itemKey("Alice"),
					},
					{
						Item: itemKey("Bob"),
					},
				}
				a.cache.Put(items[0])
				a.cache.Put(items[1])
				a.cache.Put(items[2])
				return a
			},
			resp: &etcdserverpb.RangeResponse{
				Header: &etcdserverpb.ResponseHeader{
					Revision: 13,
				},
				Kvs: []*mvccpb.KeyValue{
					{
						Key:   []byte("Alice"),
						Value: []byte("Alice"),
					},
					{
						Key:   []byte("Bob"),
						Value: []byte("Bob"),
					},
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := tc.prepareFn()
			resp, err := a.Range(context.Background(), tc.rr)
			assert.Nil(t, err, "checking error")
			assert.Equal(t, a.revision, resp.Header.Revision, "checking revision")
			assert.Equal(t, len(tc.resp.Kvs), len(resp.Kvs), "checking number of KeyValues")
			for i, kv := range tc.resp.Kvs {
				assert.Equalf(t, kv.Key, resp.Kvs[i].Key, "checking #%d key", i)
				assert.Equalf(t, kv.Value, resp.Kvs[i].Value, "checking #%d value", i)
			}
		})
	}
}
