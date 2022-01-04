package etcdadapter

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/k3s-io/kine/pkg/server"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/api7/etcd-adapter/backends/btree"
	"github.com/api7/etcd-adapter/backends/mysql"
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

	logger  *zap.Logger
	grpcSrv *grpc.Server
	httpSrv *http.Server

	eventsCh chan []*Event
	backend  server.Backend
	bridge   *bridge
}

// AdapterOptions contains fields that can control the adapter behaviors.
type AdapterOptions struct {
	Logger       *zap.Logger
	Backend      BackendKind
	MySQLOptions *mysql.Options
}

// bridge wraps the server.KVServerBridge so that we can overrides some
// methods of it to overcome the constraints.
type bridge struct {
	*server.KVServerBridge
}

// NewEtcdAdapter new an etcd adapter instance.
func NewEtcdAdapter(opts *AdapterOptions) Adapter {
	var (
		err     error
		logger  *zap.Logger
		backend server.Backend
	)
	if opts != nil && opts.Logger != nil {
		logger = opts.Logger
	} else {
		logger = zap.NewExample()
	}
	switch opts.Backend {
	case BackendBTree:
		backend = btree.NewBTreeCache(logger)
	case BackendMySQL:
		backend, err = mysql.NewMySQLCache(context.TODO(), opts.MySQLOptions)
		if err != nil {
			panic(fmt.Sprintf("failed to create mysql backend: %s", err))
		}
	default:
		panic("unknown backend")
	}

	bridge := &bridge{
		KVServerBridge: server.New(backend, ""),
	}

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
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(`{"etcdserver":"3.5.0-pre","etcdcluster":"3.5.0"}`))
	if err != nil {
		a.logger.Warn("failed to send version info",
			zap.Error(err),
		)
	}
}

func (b *bridge) Put(ctx context.Context, r *etcdserverpb.PutRequest) (*etcdserverpb.PutResponse, error) {
	rangeResp, err := b.Range(ctx, &etcdserverpb.RangeRequest{
		Key: r.Key,
	})
	if err != nil {
		return nil, err
	}

	cmp := clientv3.ModRevision(string(r.Key))
	if len(rangeResp.Kvs) > 0 {
		cmp = clientv3.Compare(cmp, "=", rangeResp.Kvs[0].ModRevision)
	} else {
		cmp = clientv3.Compare(cmp, "=", 0)
	}

	txn := &etcdserverpb.TxnRequest{
		Compare: []*etcdserverpb.Compare{
			(*etcdserverpb.Compare)(&cmp),
		},
		Success: []*etcdserverpb.RequestOp{
			{
				Request: &etcdserverpb.RequestOp_RequestPut{
					RequestPut: r,
				},
			},
		},
		Failure: []*etcdserverpb.RequestOp{
			{
				Request: &etcdserverpb.RequestOp_RequestRange{
					RequestRange: &etcdserverpb.RangeRequest{
						Key: r.Key,
					},
				},
			},
		},
	}

	resp, err := b.KVServerBridge.Txn(ctx, txn)
	if err != nil {
		return nil, err
	}
	if len(resp.Responses) == 0 {
		return nil, fmt.Errorf("broken internal put implementation")
	}
	return resp.Responses[0].GetResponsePut(), nil
}

func (b *bridge) DeleteRange(ctx context.Context, r *etcdserverpb.DeleteRangeRequest) (*etcdserverpb.DeleteRangeResponse, error) {
	// See https://github.com/k3s-io/kine/blob/c1edece777/pkg/server/delete.go#L9
	// to learn how kine decides whether this is a DELETE request.
	txn := &etcdserverpb.TxnRequest{
		Compare: []*etcdserverpb.Compare{},
		Success: []*etcdserverpb.RequestOp{
			{
				Request: &etcdserverpb.RequestOp_RequestRange{
					RequestRange: &etcdserverpb.RangeRequest{
						Key:      r.Key,
						RangeEnd: r.RangeEnd,
					},
				},
			},
			{
				Request: &etcdserverpb.RequestOp_RequestDeleteRange{
					RequestDeleteRange: r,
				},
			},
		},
	}

	resp, err := b.KVServerBridge.Txn(ctx, txn)
	if err != nil {
		return nil, err
	}
	// TODO fix the kine bug that the response type is not DeleteRange.
	rangeResp := resp.Responses[0].GetResponseRange()
	if len(resp.Responses) == 0 {
		return nil, fmt.Errorf("broken internal delete_range implementation")
	}

	deleteRangeResp := &etcdserverpb.DeleteRangeResponse{
		Header:  rangeResp.Header,
		Deleted: int64(len(rangeResp.Kvs)),
	}
	return deleteRangeResp, nil
}

// Register copies the behaviors of the b.KVServerBridge as we want to override
// some RPC implementation of ETCD.
func (b *bridge) Register(server *grpc.Server) {
	etcdserverpb.RegisterLeaseServer(server, b)
	etcdserverpb.RegisterWatchServer(server, b)
	etcdserverpb.RegisterKVServer(server, b)
	etcdserverpb.RegisterClusterServer(server, b)
	etcdserverpb.RegisterMaintenanceServer(server, b)

	hsrv := health.NewServer()
	hsrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, hsrv)
}
