package etcdadapter

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/api7/etcd-adapter/backends/btree"
	"github.com/k3s-io/kine/pkg/server"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/api7/etcd-adapter/backends"
)

// EventType is the type of event kind.
type EventType int

const (
	// EventAdd is the add event.
	EventAdd = EventType(iota + 1)
	// EventUpdate is the update event.
	EventUpdate
	// EventDelete is the delete event
	EventDelete
)

var (
	_errInternalError = status.New(codes.Internal, "internal error").Err()
)

// cacheItem wraps the backends.item with some etcd specific concepts
// like create revision and modify revision.
type cacheItem struct {
	backends.Item

	createRevision int64
	modRevision    int64
}

// MarshalLogObject implements the zapcore.ObjectMarshal interface.
// TODO performance benchmark.
func (item *cacheItem) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	oe.AddInt64("create_revision", item.createRevision)
	oe.AddInt64("modify_revision", item.modRevision)
	if err := oe.AddReflected("value", item.Item); err != nil {
		return err
	}
	return nil
}

// Event contains a bunch of entities and the type of event.
type Event struct {
	// Item is the slice of event entites.
	Items []backends.Item
	// Type is the event type.
	Type EventType
}

// itemKey implements the backends.Item interface.
type itemKey string

func (ik itemKey) Key() string {
	return string(ik)
}

func (ik itemKey) Marshal() ([]byte, error) {
	return []byte(ik), nil
}

type Adapter interface {
	// EventCh returns a send-only channel to the users, so that users
	// can feed events to Etcd Adapter. Note this is a non-buffered channel.
	EventCh() chan<- *Event
	// Serve accepts a net.Listener object and starts the Etcd V3 server.
	Serve(context.Context, net.Listener) error
	// Shutdown shuts the etcd adapter down.
	Shutdown(context.Context) error
}

type adapter struct {
	revision int64
	ctx      context.Context
	bridge   *server.KVServerBridge

	logger  *zap.Logger
	grpcSrv *grpc.Server
	httpSrv *http.Server

	eventsCh chan *Event
	backend  server.Backend

	mu sync.RWMutex
}

type AdapterOptions struct {
	logger *zap.Logger
}

// NewEtcdAdapter new an etcd adapter instance.
func NewEtcdAdapter(opts *AdapterOptions) Adapter {
	var logger *zap.Logger
	if opts != nil && opts.logger != nil {
		logger = opts.logger
	} else {
		logger = zap.NewExample()
	}
	backend := btree.NewBTreeCache(logger, nil)
	bridge := server.New(backend)
	a := &adapter{
		eventsCh: make(chan *Event),
		backend:  backend,
		bridge:   bridge,
	}
	return a
}

func (a *adapter) EventCh() chan<- *Event {
	return a.eventsCh
}

func (a *adapter) watchEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-a.eventsCh:
			for _, it := range ev.Items {
				rev := a.incrRevision()
				ci := &cacheItem{
					Item:        it,
					modRevision: rev,
				}
				switch ev.Type {
				case EventAdd:
					ci.createRevision = rev
					a.cache.Put(ci)
					a.logger.Debug("add event received",
						zap.Object("item", ci),
					)
				case EventUpdate:
					if old := a.cache.Get(it); old != nil {
						ci.createRevision = old.(*cacheItem).createRevision
						a.cache.Put(ci)
						a.logger.Debug("update event received",
							zap.Object("item", ci),
						)
					} else {
						a.logger.Error("ignore update event as object is not found from the backends",
							zap.Object("item", ci),
						)
					}
				case EventDelete:
					if old := a.cache.Get(it); old != nil {
						ci.createRevision = old.(*cacheItem).createRevision
						a.cache.Delete(ci)
						a.logger.Debug("delete event received",
							zap.Object("item", ci),
						)
					} else {
						a.logger.Error("ignore delete event as object is not found from the backends",
							zap.Object("item", ci),
						)
					}
				}

				a.sendEvent(ci, ev.Type)
			}
		}
	}
}

func (a *adapter) sendEvent(ci *cacheItem, typ EventType) {
	key := ci.Key()
	value, err := ci.Marshal()
	if err != nil {
		a.logger.Error("failed to marshal item, ignore it",
			zap.Error(err),
			zap.Object("item", ci),
		)
		return
	}
	event := &mvccpb.Event{
		Kv: &mvccpb.KeyValue{
			Key:            []byte(key),
			CreateRevision: ci.createRevision,
			ModRevision:    ci.modRevision,
			Value:          value,
		},
	}
	switch typ {
	case EventAdd, EventUpdate:
		event.Type = mvccpb.PUT
	case EventDelete:
		event.Type = mvccpb.DELETE
	}

	ws := a.watchers.watcherSetByKey(ci.Key())
	for w := range ws {
		go func(w *watcher) {
			w.eventCh <- event
		}(w)
	}
}

func (a *adapter) incrRevision() int64 {
	old := atomic.LoadInt64(&a.revision)
	for {
		if atomic.CompareAndSwapInt64(&a.revision, old, old+1) {
			return old + 1
		}
	}
}

func (a *adapter) showVersion(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(`{"etcdserver":"3.5.0-pre","etcdcluster":"3.5.0"}`))
	if err != nil {
		a.logger.Warn("failed to send version info",
			zap.Error(err),
		)
	}
}
