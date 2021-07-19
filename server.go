// Copyright api7.ai
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package etcdadapter

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/soheilhy/cmux"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	etcdservergw "go.etcd.io/etcd/api/v3/etcdserverpb/gw"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func (a *adapter) Serve(ctx context.Context, l net.Listener) error {
	a.ctx = ctx

	m := cmux.New(l)
	grpcl := m.Match(cmux.HTTP2())
	httpl := m.Match(cmux.HTTP1Fast())

	kep := keepalive.EnforcementPolicy{
		MinTime: 15 * time.Second,
	}
	kp := keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute,
		Timeout:           10 * time.Second,
	}

	grpcSrv := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kep),
		grpc.KeepaliveParams(kp),
	)
	a.grpcSrv = grpcSrv
	//etcdserverpb.RegisterKVServer(grpcSrv, a)
	//etcdserverpb.RegisterWatchServer(grpcSrv, a)
	if gwmux, err := a.registerGateway(l.Addr().String()); err != nil {
		return err
	} else {
		mux := http.NewServeMux()
		mux.Handle(
			"/v3/",
			wsproxy.WebsocketProxy(
				gwmux,
				wsproxy.WithRequestMutator(
					func(incoming *http.Request, outgoing *http.Request) *http.Request {
						outgoing.Method = "POST"
						return outgoing
					},
				),
			),
		)
		mux.HandleFunc("/version", a.showVersion)
		a.httpSrv = &http.Server{
			Handler: mux,
		}
	}

	go a.watchEvents(ctx)

	go func() {
		if err := a.httpSrv.Serve(httpl); err != nil && !strings.Contains(err.Error(), "mux: listener closed") {
			a.logger.Error("http server serve failure",
				zap.Error(err),
			)
		}
	}()

	go func() {
		if err := grpcSrv.Serve(grpcl); err != nil {
			a.logger.Error("grpc server serve failure",
				zap.Error(err),
			)
		}
	}()

	if err := m.Serve(); err != nil && !reasonableFailure(err) {
		return err
	}

	return nil
}

func (a *adapter) Shutdown(ctx context.Context) error {
	a.grpcSrv.GracefulStop()
	if err := a.httpSrv.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

// registerGateway registers a gRPC gateway server for etcd adapter, as some components
// might not support gRPC protocol, it's better to support the HTTP Restful protocol.
func (a *adapter) registerGateway(addr string) (*gatewayruntime.ServeMux, error) {
	a.logger.Info("register grpc gateway")
	grpcConn, err := grpc.DialContext(a.ctx, addr,
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	gwmux := gatewayruntime.NewServeMux()
	if err := etcdservergw.RegisterKVHandler(a.ctx, gwmux, grpcConn); err != nil {
		return nil, err
	}
	if err := etcdservergw.RegisterWatchHandler(a.ctx, gwmux, grpcConn); err != nil {
		return nil, err
	}
	go func() {
		<-a.ctx.Done()
		if err := grpcConn.Close(); err != nil {
			a.logger.Error("failed to close local gateway grpc conn",
				zap.Error(err),
			)
		}
	}()
	return gwmux, nil
}

func reasonableFailure(err error) bool {
	if err == http.ErrServerClosed {
		return true
	}
	if strings.Contains(err.Error(), "mux: listener closed") {
		return true
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}
