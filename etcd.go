package etcdadapter

import (
	"context"

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
			for _, item := range ev.Items {
				switch ev.Type {
				case EventAdd:
					a.cache.Put(item)
				case EventUpdate:
					a.cache.Put(item)
				case EventDelete:
					a.cache.Delete(item)
				}
			}
		}
	}
}
