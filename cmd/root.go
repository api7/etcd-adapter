/*
Copyright © 2022 API7.ai

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/hot123s/dsn"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/api7/etcd-adapter/internal/adapter"
	"github.com/api7/etcd-adapter/internal/backends/mysql"
	"github.com/api7/etcd-adapter/internal/config"
	"github.com/api7/etcd-adapter/internal/utils"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "etcd-adapter",
	Short: "The bridge between etcd protocol and other storage backends.",
	Run: func(cmd *cobra.Command, args []string) {
		// initialize configuration
		err := config.Init(configFile)
		if err != nil {
			return
		}

		dsn, err := dsn.Parse(config.DSN)
		if err != nil {
			return
		}

		var adapterBackend adapter.BackendKind
		switch dsn.Protocol {
		case "mysql":
			adapterBackend = adapter.BackendMySQL
		default:
			adapterBackend = adapter.BackendBTree
		}

		// bootstrap etcd adapter
		opts := &adapter.AdapterOptions{
			Logger:  utils.GetLogger(),
			Backend: adapterBackend,
			MySQLOptions: &mysql.Options{
				DSN: config.DSN,
			},
		}
		a := adapter.NewEtcdAdapter(opts)

		ln, err := net.Listen("tcp", net.JoinHostPort(config.Server.Host, config.Server.Port))
		if err != nil {
			panic(err)
		}
		go func() {
			if err := a.Serve(context.Background(), ln); err != nil {
				panic(err)
			}
		}()

		// graceful exit
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-quit:
			err := a.Shutdown(context.TODO())
			_ = utils.GetLogger().Sync()
			if err != nil {
				utils.GetLogger().Error("An error occurred while exiting", zap.Error(err))
				return
			}
			utils.GetLogger().Info("See you next time!")
		}
	},
}

// Execute bootstrap root command.
func Execute() {
	// declare flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file")

	// execute root command
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
