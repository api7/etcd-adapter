ETCD Adapter
============

ETCD Adapter mimics the [ETCD V3 APIs](https://etcd.io/docs/v3.3/rfc/) best effort. It incorporates the [kine](https://github.com/k3s-io/kine) as the 
Server side implementation, and it develops a totally in-memory watchable backend.

**Not all features in ETCD V3 APIs supported**, this is designed for [Apache APISIX](https://apisix.apache.org), so it's inherently not a generic solution.

How to use it
-------------

It's easy to create an etcd adapter instance. The use of this instance contains two parts, the first thing is you should prepare a network listener and let it run. ETCD adapter
launches a gRPC service (with a built-in [gRPC Gateway](https://github.com/grpc-ecosystem/grpc-gateway), so that HTTP Restful requests can also be handled), you can try to access this
service with the [etcdctl](https://github.com/etcd-io/etcd/tree/main/etcdctl) tool.

The another thing is you should feed some events to this instance, so that data changes can be applied to the instance and ultimately it affects the end-user of this gRPC service.

```go
package main

import (
        "context"
        "fmt"
        "math/rand"
        "net"
        "time"

        adapter "github.com/api7/etcd-adapter"
)

func init() {
        rand.Seed(time.Now().UnixNano())
}

func main() {
        a := adapter.NewEtcdAdapter(nil)
        ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Hour)
        defer cancel()
        go produceEvents(ctx, a)

        ln, err := net.Listen("tcp", "127.0.0.1:12379")
        if err != nil {
                panic(err)
        }
        go func() {
                if err := a.Serve(context.Background(), ln); err != nil {
                        panic(err)
                }
        }()
        <-ctx.Done()
        if err := a.Shutdown(context.TODO()); err != nil {
                panic(err)
        }
}

func produceEvents(ctx context.Context, a adapter.Adapter) {
        ticker := time.NewTicker(time.Second)
        for {
                select {
                case <-ctx.Done():
                        return
                case <-ticker.C:
                        break
                }
                event := &adapter.Event{
                        Type: adapter.EventType(rand.Intn(3) + 1),
                        Key:  fmt.Sprintf("/key/part/%d", rand.Int()),
                }
                if event.Type != adapter.EventDelete {
                        event.Value = []byte(fmt.Sprintf("value-%d", rand.Int()))
                }
                a.EventCh() <- []*adapter.Event{event}
        }
}
```

The above example shows a simple usage about the etcd adapter.

**Note, get keys by prefix constrained strictly as the key format has to be path-like**, for instance, keys can be `/apisix/routes/1`, `apisix/upstreams/2`, and you can get them with
the prefix `/apisix`, or `/apisix/routes`, `/apisix/upstreams` perspective.
