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

package mysql

import (
	"context"

	"github.com/k3s-io/kine/pkg/drivers/generic"
	mysqldriver "github.com/k3s-io/kine/pkg/drivers/mysql"
	"github.com/k3s-io/kine/pkg/server"
	"github.com/k3s-io/kine/pkg/tls"
)

// Options contains settings for controlling the connection to MySQL.
type Options struct {
	DSN      string
	ConnPool generic.ConnectionPoolConfig
}

type mysqlCache struct {
	server.Backend
}

// NewMySQLCache returns a server.Backend interface which was implemented with
// the MySQL backend. The first argument `ctx` is used to control the lifecycle of
// mysql connection pool.
func NewMySQLCache(ctx context.Context, options *Options) (server.Backend, error) {
	backend, err := mysqldriver.New(ctx, options.DSN, tls.Config{}, options.ConnPool)
	if err != nil {
		return nil, err
	}
	mc := &mysqlCache{
		Backend: backend,
	}
	return mc, nil
}

func (m *mysqlCache) Start(ctx context.Context) error {
	return m.Backend.Start(ctx)
}
