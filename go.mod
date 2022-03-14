module github.com/api7/etcd-adapter

go 1.16

require (
	github.com/api7/gopkg v0.1.2
	github.com/google/btree v1.0.1
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/k3s-io/kine v0.8.1
	github.com/soheilhy/cmux v0.1.5
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.0
	github.com/stretchr/testify v1.7.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802
	go.etcd.io/etcd v3.3.27+incompatible // indirect
	go.etcd.io/etcd/api/v3 v3.5.2
	go.etcd.io/etcd/client/v3 v3.5.2
	go.uber.org/zap v1.19.1
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	google.golang.org/grpc v1.42.0
)

replace go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738 => go.etcd.io/etcd v3.3.27+incompatible
