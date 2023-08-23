package sqlite

import (
	"context"
	"github.com/k3s-io/kine/pkg/drivers/generic"
	"github.com/k3s-io/kine/pkg/drivers/sqlite"
	"github.com/k3s-io/kine/pkg/server"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"os"
)

// Options contains settings for controlling the connection to MySQL.
type Options struct {
	DSN      string
	ConnPool generic.ConnectionPoolConfig
	Prom     prometheus.Registerer
}

type sqliteCache struct {
	server.Backend
}

func init() {
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

func NewSQLiteCache(ctx context.Context, options *Options) (server.Backend, error) {
	log.Debug("NEW SQLite ...")
	dsn := options.DSN
	log.Debug("DNS:", dsn)
	backend, err := sqlite.New(ctx, dsn, options.ConnPool, options.Prom)
	if err != nil {
		return nil, err
	}
	sc := &sqliteCache{
		Backend: backend,
	}
	return sc, nil
}

func (s *sqliteCache) Start(ctx context.Context) error {
	log.Debug("SQLite Start ...")
	return s.Backend.Start(ctx)
}

func (s *sqliteCache) Get(ctx context.Context, key string, rangeEnd string, limit int64, revision int64) (int64, *server.KeyValue, error) {
	log.Debug("SQLite Get KEY:", key)
	return s.Backend.Get(ctx, key, rangeEnd, limit, revision)
}

func (s *sqliteCache) Create(ctx context.Context, key string, value []byte, lease int64) (int64, error) {
	log.Debug("SQLite Create KEY:", key, " VAL: ", value)
	return s.Backend.Create(ctx, key, value, lease)
}

func (s *sqliteCache) Delete(ctx context.Context, key string, revision int64) (int64, *server.KeyValue, bool, error) {
	log.Debug("SQLite Delete KEY:", key)
	return s.Backend.Delete(ctx, key, revision)
}

func (s *sqliteCache) List(ctx context.Context, prefix string, startKey string, limit int64, revision int64) (int64, []*server.KeyValue, error) {
	log.Debug("SQLite List PREFIX:", prefix, " STARTKEY:", startKey)
	return s.Backend.List(ctx, prefix, startKey, limit, revision)
}

func (s *sqliteCache) Count(ctx context.Context, prefix string) (int64, int64, error) {
	log.Debug("SQLite Count PREFIX:", prefix)
	return s.Backend.Count(ctx, prefix)
}
func (s *sqliteCache) Update(ctx context.Context, key string, value []byte, revision int64, lease int64) (int64, *server.KeyValue, bool, error) {
	log.Debug("SQLite Update KEY:", key, " VAL:", value)
	return s.Backend.Update(ctx, key, value, revision, lease)
}

func (s *sqliteCache) Watch(ctx context.Context, key string, revision int64) <-chan []*server.Event {
	log.Debug("SQLite Watch KEY:", key)
	return s.Backend.Watch(ctx, key, revision)
}

func (s *sqliteCache) DbSize(ctx context.Context) (int64, error) {
	log.Debug("SQLite DbSize ..")
	return s.Backend.DbSize(ctx)
}
