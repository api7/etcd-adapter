package etcdserver

import (
	"github.com/k3s-io/kine/pkg/server"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type EtcdServer struct {
	*server.KVServerBridge
	backend server.Backend
}

func NewEtcdServer(backend server.Backend) *EtcdServer {
	return &EtcdServer{
		KVServerBridge: server.New(backend, ""),
		backend:        backend,
	}
}

func (svr *EtcdServer) Register(server *grpc.Server) {
	etcdserverpb.RegisterWatchServer(server, svr)
	etcdserverpb.RegisterKVServer(server, svr)

	// Bridge comes from the implementation of kine
	etcdserverpb.RegisterLeaseServer(server, svr)
	etcdserverpb.RegisterClusterServer(server, svr)
	etcdserverpb.RegisterMaintenanceServer(server, svr)

	hsrv := health.NewServer()
	hsrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, hsrv)
}
