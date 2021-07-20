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
	"fmt"
	"io"

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.uber.org/zap"
)

func (a *adapter) Watch(stream etcdserverpb.Watch_WatchServer) error {
	a.logger.Debug("watch stream built")
	ctx, cancel := context.WithCancel(stream.Context())
	errCh := make(chan error, 1)
	go a.watchOnWire(ctx, stream, errCh)

	for {
		select {
		case err := <-errCh:
			cancel()
			return err
		case <-stream.Context().Done():
			a.logger.Debug("client closed watch stream prematurely",
				zap.Error(stream.Context().Err()),
			)
			cancel()
			return nil
		}
	}
}

func (a *adapter) watchOnWire(ctx context.Context, stream etcdserverpb.Watch_WatchServer, errCh chan error) {
	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				errCh <- nil
				return
			}
			a.logger.Warn("watch stream recv failed", zap.Error(err))
			errCh <- err
			return
		}
		a.logger.Debug("watch request arrived",
			zap.String("request", req.String()),
		)

		switch uv := req.RequestUnion.(type) {
		case *etcdserverpb.WatchRequest_CreateRequest:
			if uv.CreateRequest == nil {
				continue
			}
			wa := &watcher{
				minRev:  uv.CreateRequest.StartRevision,
				key:     uv.CreateRequest.Key,
				end:     uv.CreateRequest.RangeEnd,
				a:       a,
				stream:  stream,
				errCh:   errCh,
				eventCh: make(chan *mvccpb.Event),
			}
			a.watchers.add(wa)
			a.logger.Debug("added watcher",
				zap.Int64("id", wa.id),
			)
			a.watcherCancelMu.Lock()
			// TODO we may also cancel watchers once
			// if the connection was aborted.
			a.watcherCancel[wa.id] = func() (ok bool) {
				ok = a.watchers.delete(wa)
				return
			}
			go wa.listen(ctx)
			a.watcherCancelMu.Unlock()

			if err := wa.initialSend(); err != nil {
				errCh <- err
				return
			}

		case *etcdserverpb.WatchRequest_CancelRequest:
			if uv.CancelRequest == nil {
				continue
			}
			id := uv.CancelRequest.WatchId
			a.watcherCancelMu.Lock()
			cancelFunc, ok := a.watcherCancel[id]
			delete(a.watcherCancel, id)
			a.watcherCancelMu.Unlock()
			if !ok || !cancelFunc() {
				errCh <- fmt.Errorf("unknown watch id %d", id)
				return
			}
		}
	}
}
