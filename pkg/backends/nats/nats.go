package nats

import (
	"context"
	"github.com/api7/gopkg/pkg/log"
	"github.com/k3s-io/kine/pkg/drivers/generic"
	natsdriver "github.com/k3s-io/kine/pkg/drivers/nats"
	"github.com/k3s-io/kine/pkg/server"
	"github.com/k3s-io/kine/pkg/tls"
	"go.uber.org/zap"
)

// Options contains settings for controlling the connection to MySQL.
type Options struct {
	DSN      string
	ConnPool generic.ConnectionPoolConfig
}

type natsCache struct {
	server.Backend
}

// NewNATSCache returns a server.Backend interface which was implemented with
// the NATS embedded backend. The first argument `ctx` is used to control the lifecycle of
// nats connection pool.
func NewNATSCache(ctx context.Context, options *Options) (server.Backend, error) {
	dsn := options.DSN
	backend, err := natsdriver.New(ctx, dsn, tls.Config{})
	if err != nil {
		log.Errorw("ERR: %w", zap.Error(err))
		return nil, err
	}
	nc := &natsCache{
		Backend: backend,
	}
	return nc, nil
}

// None of below needs to be implemented .. only for debug sake without modifying inside kine ..

//func (n *natsCache) Start(ctx context.Context) error {
//	log.Info("START NATS \n\n")
//	return n.Backend.Start(ctx)
//}

func (n *natsCache) Get(ctx context.Context, key string, rangeEnd string, limit int64, revision int64) (int64, *server.KeyValue, error) {
	log.Info("GET NATS KEY:", key, " RANGE:", rangeEnd, " LIMIT:", limit, " REV:", revision, "\n\n")
	return n.Backend.Get(ctx, key, rangeEnd, limit, revision)
}

func (n *natsCache) Create(ctx context.Context, key string, value []byte, lease int64) (int64, error) {
	log.Info("GET NATS KEY:", key, " VAL:", string(value), " LEASE:", lease)
	return n.Backend.Create(ctx, key, value, lease)
}

// func (n *natsCache) Delete(ctx context.Context, key string, revision int64) (int64, *KeyValue, bool, error)
func (n *natsCache) List(ctx context.Context, prefix string, startKey string, limit int64, revision int64) (int64, []*server.KeyValue, error) {
	log.Info("LIST NATS PREFIX:", prefix, " SKEY:", startKey, " LIMIT:", limit, " REV:", revision)
	return n.Backend.List(ctx, prefix, startKey, limit, revision)
}

//Count(ctx context.Context, prefix string) (int64, int64, error)
//Update(ctx context.Context, key string, value []byte, revision, lease int64) (int64, *KeyValue, bool, error)
//Watch(ctx context.Context, key string, revision int64) WatchResult
//DbSize(ctx context.Context) (int64, error)
//CurrentRevision(ctx context.Context) (int64, error)
