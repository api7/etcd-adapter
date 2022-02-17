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

package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/nettest"
)

func TestShowVersion(t *testing.T) {
	w := httptest.NewRecorder()
	a := NewEtcdAdapter(nil).(*adapter)
	a.showVersion(w, nil)

	assert.Equal(t, http.StatusOK, w.Code, "checking status code")
	assert.Equal(t, "{\"etcdserver\":\"3.5.0-pre\",\"etcdcluster\":\"3.5.0\"}", w.Body.String())
}

func TestEtcdAdapter(t *testing.T) {
	a := NewEtcdAdapter(nil)

	ln, err := nettest.NewLocalListener("tcp")
	assert.Nil(t, err, "checking listener creating error")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := a.Serve(ctx, ln)
		assert.Nil(t, err, "checking serve returning error")
	}()

	events := []*Event{
		{
			Key:   "/apisix/routes/1",
			Value: []byte("123"),
			Type:  EventAdd,
		},
		{
			Key:   "/apisix/routes/2",
			Value: []byte("456"),
			Type:  EventAdd,
		},
	}

	a.EventCh() <- events
	time.Sleep(500 * time.Millisecond)

	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{ln.Addr().String()},
	})
	assert.Nil(t, err, "creating etcd client")
	resp, err := client.Get(context.Background(), "/apisix/routes/1")
	assert.Nil(t, err, "checking error")
	assert.Len(t, resp.Kvs, 1, "checking number of kvs")
	assert.Equal(t, []byte("123"), resp.Kvs[0].Value)
	assert.Equal(t, "/apisix/routes/1", string(resp.Kvs[0].Key))

	resp, err = client.Get(context.Background(), "/apisix/routes/2")
	assert.Nil(t, err, "checking error")
	assert.Len(t, resp.Kvs, 1, "checking number of kvs")
	assert.Equal(t, []byte("456"), resp.Kvs[0].Value)
	assert.Equal(t, "/apisix/routes/2", string(resp.Kvs[0].Key))

	resp, err = client.Get(context.Background(), "/apisix/routes/3")
	assert.Nil(t, err, "checking error")
	assert.Len(t, resp.Kvs, 0, "checking number of kvs")

	resp, err = client.Get(context.Background(), "/apisix/routes", clientv3.WithPrefix())
	assert.Nil(t, err, "checking error")
	assert.Len(t, resp.Kvs, 2, "checking number of kvs")
	assert.Equal(t, string(resp.Kvs[0].Key), "/apisix/routes/1")
	assert.Equal(t, string(resp.Kvs[1].Key), "/apisix/routes/2")

	events = []*Event{
		{
			Key:   "/apisix/routes/1",
			Value: []byte("456"),
			Type:  EventUpdate,
		},
		{
			Key:  "/apisix/routes/2",
			Type: EventDelete,
		},
	}

	a.EventCh() <- events
	time.Sleep(500 * time.Millisecond)

	resp, err = client.Get(context.Background(), "/apisix/routes/1")
	assert.Nil(t, err, "checking error")
	assert.Len(t, resp.Kvs, 1, "checking number of kvs")
	assert.Equal(t, []byte("456"), resp.Kvs[0].Value)
	assert.Equal(t, "/apisix/routes/1", string(resp.Kvs[0].Key))

	resp, err = client.Get(context.Background(), "/apisix/routes/2")
	assert.Nil(t, err, "checking error")
	assert.Len(t, resp.Kvs, 0, "checking number of kvs")

	err = a.Shutdown(ctx)
	assert.Nil(t, err, "shutting down")
	cancel()
}

func TestEtcdAdapterWatch(t *testing.T) {
	events := []*Event{
		{
			Key:   "/apisix/routes/1",
			Value: []byte("123"),
			Type:  EventAdd,
		},
		{
			Key:   "/apisix/routes/1",
			Value: []byte("456"),
			Type:  EventUpdate,
		},
	}

	a := NewEtcdAdapter(nil)

	ln, err := nettest.NewLocalListener("tcp")
	assert.Nil(t, err, "checking listener creating error")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := a.Serve(ctx, ln)
		assert.Nil(t, err, "checking serve returning error")
	}()

	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{ln.Addr().String()},
	})
	assert.Nil(t, err, "creating etcd client")

	ch := client.Watch(ctx, "/apisix/routes", clientv3.WithPrefix())

	a.EventCh() <- events

	resp := <-ch
	assert.Len(t, resp.Events, 2)
	assert.Equal(t, resp.Events[0].Type, clientv3.EventTypePut)
	assert.Equal(t, string(resp.Events[0].Kv.Key), "/apisix/routes/1")
	assert.Equal(t, resp.Events[0].Kv.Value, []byte("123"))
	assert.Equal(t, resp.Events[1].Type, clientv3.EventTypePut)
	assert.Equal(t, string(resp.Events[1].Kv.Key), "/apisix/routes/1")
	assert.Equal(t, resp.Events[1].Kv.Value, []byte("456"))

	events = []*Event{
		{
			Key:  "/apisix/routes/1",
			Type: EventDelete,
		},
	}
	a.EventCh() <- events
	resp = <-ch
	assert.Len(t, resp.Events, 1)
	assert.Equal(t, resp.Events[0].Type, clientv3.EventTypeDelete)
	assert.Equal(t, string(resp.Events[0].PrevKv.Key), "/apisix/routes/1")
	assert.Equal(t, resp.Events[0].PrevKv.Value, []byte("456"))

	cancel()
}
