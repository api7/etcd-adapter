package adapter

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/api7/gopkg/pkg/log"
	"github.com/k3s-io/kine/pkg/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

// BackendKind is the type of backend.
type BackendKind int

const (
	// BackendBTree indicates the btree-based backend.
	BackendBTree = BackendKind(iota)
	// BackendMySQL indicates the mysql-based backend.
	BackendMySQL
)

// Event contains a bunch of entities and the type of event.
type Event struct {
	// Key is the object key.
	Key string
	// Value is the serialized data.
	Value []byte
	// Type is the event type.
	Type EventType
}

type Adapter interface {
	// EventCh returns a send-only channel to the users, so that users
	// can feed events to Etcd Adapter. Note this is a non-buffered channel.
	EventCh() chan<- []*Event
	// Serve accepts a net.Listener object and starts the Etcd V3 server.
	Serve(context.Context, net.Listener) error
	// Shutdown shuts the etcd adapter down.
	Shutdown(context.Context) error
}

type adapter struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger  *log.Logger
	grpcSrv *grpc.Server
	httpSrv *http.Server

	eventsCh chan []*Event
	backend  server.Backend
	bridge   *server.KVServerBridge
}

// NewEtcdAdapter new an etcd adapter instance.
func NewEtcdAdapter(backend server.Backend, logger *log.Logger) Adapter {
	bridge := server.New(backend, "")
	a := &adapter{
		logger:   logger,
		eventsCh: make(chan []*Event),
		backend:  backend,
		bridge:   bridge,
	}
	return a
}

func (a *adapter) EventCh() chan<- []*Event {
	return a.eventsCh
}

func (a *adapter) watchEvents(ctx context.Context) {
	for {
		var events []*Event
		select {
		case <-ctx.Done():
			return
		case events = <-a.eventsCh:
			break
		}
		if len(events) > 0 {
			for _, ev := range events {
				// TODO we may use separate goroutines to handle events so that
				// this main cycle won't be blocked, but the concurrency might cause
				// the handling order is unpredictable, so this is a judgement call.
				switch ev.Type {
				case EventAdd:
					a.handleAddEvent(ctx, ev)
				case EventUpdate:
					a.handleUpdateEvent(ctx, ev)
				case EventDelete:
					a.handleDeleteEvent(ctx, ev)
				}
			}
		}
	}
}

func (a *adapter) handleAddEvent(ctx context.Context, ev *Event) {
	rev, err := a.backend.Create(ctx, ev.Key, ev.Value, 0)
	if err != nil {
		a.logger.Error("failed to create object, ignore it",
			zap.Error(err),
			zap.Int64("revision", rev),
			zap.String("key", ev.Key),
		)
	} else {
		a.logger.Info("created object",
			zap.Int64("revision", rev),
			zap.String("key", ev.Key),
		)
	}
}

func (a *adapter) handleUpdateEvent(ctx context.Context, ev *Event) {
	for {
		rev, prevKV, err := a.backend.Get(ctx, ev.Key, 0)
		if err != nil {
			a.logger.Error("failed to get object (during update event), ignore it",
				zap.Error(err),
				zap.Int64("revision", rev),
				zap.String("key", ev.Key),
			)
			return
		}
		if prevKV == nil {
			a.logger.Error("object not found (during update event), ignore it",
				zap.Int64("revision", rev),
				zap.String("key", ev.Key),
			)
			return
		}
		rev, prev, ok, err := a.backend.Update(ctx, ev.Key, ev.Value, prevKV.ModRevision, 0)
		if err != nil || prev == nil {
			if prev == nil {
				err = errors.New("object not found")
			}
			a.logger.Error("failed to update object, ignore it",
				zap.Error(err),
				zap.Int64("revision", rev),
				zap.String("key", ev.Key),
			)
			return
		}
		if ok {
			a.logger.Info("updated object",
				zap.Int64("revision", rev),
				zap.String("key", ev.Key),
			)
			return
		}
		// Update was failed due to race conditions.
		a.logger.Debug("object update was failed, retry it",
			zap.Int64("revision", rev),
			zap.String("key", ev.Key),
		)
	}
}

func (a *adapter) handleDeleteEvent(ctx context.Context, ev *Event) {
	for {
		rev, prevKV, err := a.backend.Get(ctx, ev.Key, 0)
		if err != nil {
			a.logger.Error("failed to get object (during delete event), ignore it",
				zap.Error(err),
				zap.Int64("revision", rev),
				zap.String("key", ev.Key),
			)
			return
		}
		if prevKV == nil {
			a.logger.Error("object not found (during delete event), ignore it",
				zap.Int64("revision", rev),
				zap.String("key", ev.Key),
			)
			return
		}
		rev, prev, ok, err := a.backend.Delete(ctx, ev.Key, prevKV.ModRevision)
		if err != nil || prev == nil {
			if prev == nil {
				err = errors.New("object not found")
			}
			a.logger.Error("failed to delete object, ignore it",
				zap.Error(err),
				zap.Int64("revision", rev),
				zap.String("key", ev.Key),
			)
			return
		}
		if ok {
			a.logger.Info("deleted object",
				zap.Int64("revision", rev),
				zap.String("key", ev.Key),
			)
			return
		}
		// Delete was failed due to race conditions.
		a.logger.Debug("object delete was failed, retry it",
			zap.Int64("revision", rev),
			zap.String("key", ev.Key),
		)
	}
}

func (a *adapter) showVersion(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(`{"etcdserver":"3.5.0","etcdcluster":"3.5.0"}`))
	if err != nil {
		a.logger.Warn("failed to send version info",
			zap.Error(err),
		)
	}
}
