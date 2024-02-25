package etcdserver

import (
	"context"
	"io"
	"sync"
	"sync/atomic"

	"github.com/api7/gopkg/pkg/log"
	"github.com/k3s-io/kine/pkg/server"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.uber.org/zap"
)

var _ etcdserverpb.WatchServer = (*EtcdServer)(nil)

var watchID int64

func (s *EtcdServer) Watch(ws etcdserverpb.Watch_WatchServer) (err error) {
	w := watcher{
		stream:  ws,
		backend: s.backend,
		watches: map[int64]func(){},
	}

	errchan := make(chan error, 1)

	go func() {
		if rerr := w.Start(); rerr != nil {
			errchan <- rerr
		}
	}()

	select {
	case err = <-errchan:
		if err == context.Canceled {
			err = rpctypes.ErrGRPCWatchCanceled
		}
	case <-ws.Context().Done():
		err = ws.Context().Err()
		if err == context.Canceled {
			err = rpctypes.ErrGRPCWatchCanceled
		}
	}
	w.Close()
	log.Debug("watcher closed", zap.Error(err))
	return err
}

type watcher struct {
	sync.Mutex

	wg      sync.WaitGroup
	backend server.Backend
	stream  etcdserverpb.Watch_WatchServer
	watches map[int64]func()
}

func (w *watcher) Start() error {
	for {
		msg, err := w.stream.Recv()
		log.Debugw("watch recv", zap.Any("request", msg), zap.Error(err))
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if msg.GetCreateRequest() != nil {
			w.HandleWatchCreateRequest(w.stream.Context(), msg.GetCreateRequest())
		} else if msg.GetCancelRequest() != nil {
			w.Cancel(msg.GetCancelRequest().WatchId, nil)
		}
	}
}

func (w *watcher) HandleWatchCreateRequest(ctx context.Context, r *etcdserverpb.WatchCreateRequest) {
	w.Lock()
	defer w.Unlock()

	ctx, cancel := context.WithCancel(ctx)

	id := atomic.AddInt64(&watchID, 1)
	w.watches[id] = cancel

	key := string(r.Key)

	log.Debugw("WATCH START", zap.Int64("watch_id", id), zap.String("key", key), zap.Int64("revision", r.StartRevision))

	if err := w.stream.Send(&etcdserverpb.WatchResponse{
		Header:  &etcdserverpb.ResponseHeader{},
		Created: true,
		WatchId: id,
	}); err != nil {
		w.Cancel(id, err)
		log.Debugw("WATCH CLOSE", zap.Int64("watch_id", id), zap.String("key", key))
		return
	}
	w.wg.Add(1)
	go w.WatchableBackend(ctx, id, key, r.StartRevision)
}

func (w *watcher) WatchableBackend(ctx context.Context, watchID int64, key string, revision int64) {
	defer w.wg.Done()
	for events := range w.backend.Watch(ctx, key, revision).Events {
		if len(events) == 0 {
			break
		}
		log.Debugw("WATCH SEND", zap.Int64("watch_id", watchID), zap.String("key", key), zap.Any("events", events))
		if err := w.stream.Send(&etcdserverpb.WatchResponse{
			Header:  txnHeader(events[len(events)-1].KV.ModRevision),
			WatchId: watchID,
			Events:  toEvents(events...),
		}); err != nil {
			w.Cancel(watchID, err)
			log.Warn("WATCH CLOSE", zap.String("key", key), zap.Int64("id", watchID), zap.Error(err))
			break
		}
	}
	w.Cancel(watchID, nil)
	log.Debugw("WATCH CLOSE", zap.String("key", key), zap.Int64("id", watchID))
}

func toEvents(events ...*server.Event) []*mvccpb.Event {
	ret := make([]*mvccpb.Event, 0, len(events))
	for _, e := range events {
		ret = append(ret, toEvent(e))
	}
	return ret
}

func toEvent(event *server.Event) *mvccpb.Event {
	e := &mvccpb.Event{
		Kv:     toKV(event.KV),
		PrevKv: toKV(event.PrevKV),
	}
	if event.Delete {
		e.Type = mvccpb.DELETE
	} else {
		e.Type = mvccpb.PUT
	}

	return e
}

func (w *watcher) Cancel(watchID int64, err error) {
	w.Lock()
	if cancel, ok := w.watches[watchID]; ok {
		cancel()
		delete(w.watches, watchID)
	}
	w.Unlock()

	reason := ""
	if err != nil {
		reason = err.Error()
	}
	log.Debugw("WATCH CANCEL", zap.Int64("watch_id", watchID), zap.String("reason", reason))
	serr := w.stream.Send(&etcdserverpb.WatchResponse{
		Header:       &etcdserverpb.ResponseHeader{},
		Canceled:     true,
		CancelReason: "watch closed",
		WatchId:      watchID,
	})
	if serr != nil && err != nil {
		log.Errorw("WATCH Failed to send cancel response", zap.Int64("watch_id", watchID), zap.Error(err), zap.Error(serr))
	}
}

func (w *watcher) Close() {
	w.Lock()
	for _, v := range w.watches {
		v()
	}
	w.Unlock()
	w.wg.Wait()
}

func toKV(kv *server.KeyValue) *mvccpb.KeyValue {
	if kv == nil {
		return nil
	}
	return &mvccpb.KeyValue{
		Key:            []byte(kv.Key),
		Value:          kv.Value,
		Lease:          kv.Lease,
		CreateRevision: kv.CreateRevision,
		ModRevision:    kv.ModRevision,
	}
}

func txnHeader(rev int64) *etcdserverpb.ResponseHeader {
	return &etcdserverpb.ResponseHeader{
		Revision: rev,
	}
}
