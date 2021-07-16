package etcdadapter

import (
	"context"
	"sync/atomic"

	"github.com/api7/etcd-adapter/cache"
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

type cacheItem struct {
	cache.Item

	createRevision int64
	modRevision    int64
}

// Event contains a bunch of entities and the type of event.
type Event struct {
	// Item is the slice of event entites.
	Items []cache.Item
	// Type is the event type.
	Type EventType
}

type Adapter interface {
	// EventCh returns a send-only channel to the users, so that users
	// can feed events to Etcd Adapter. Note this is a non-buffered channel.
	EventCh() chan<- *Event
}

type adapter struct {
	revision int64

	eventsCh chan *Event
	cache    cache.Cache
}

// NewEtcdAdapter new an etcd adapter instance.
func NewEtcdAdapter() Adapter {
	a := &adapter{
		eventsCh: make(chan *Event),
		cache:    cache.NewBTreeCache(),
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
				case EventUpdate:
					if old := a.cache.Get(it); old != nil {
						ci.createRevision = old.(*cacheItem).createRevision
						a.cache.Put(ci)
					}
				case EventDelete:
					if old := a.cache.Get(it); old != nil {
						ci.createRevision = old.(*cacheItem).createRevision
						a.cache.Delete(ci)
					}
				}
				// TODO pass ci to etcd server.
			}
		}
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
