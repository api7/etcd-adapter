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
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/api7/gopkg/pkg/log"
	"github.com/k3s-io/kine/pkg/server"
	"github.com/spf13/cobra"

	"github.com/api7/etcd-adapter/internal/adapter"
	"github.com/api7/etcd-adapter/internal/backends/btree"
	"github.com/api7/etcd-adapter/internal/backends/mysql"
	"github.com/api7/etcd-adapter/internal/config"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "etcd-adapter",
	Short: "The bridge between etcd protocol and other storage backends.",
	Run: func(cmd *cobra.Command, args []string) {
		// initialize logger
		logger, err := log.NewLogger()

		// initialize configuration
		err = config.Init(configFile, logger)
		if err != nil {
			return
		}

		// initialize backend
		var backend server.Backend
		switch config.Config.DataSource.Type {
		case "mysql":
			mysqlConfig := config.Config.DataSource.MySQL
			backend, err = mysql.NewMySQLCache(context.TODO(), &mysql.Options{
				DSN: fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", mysqlConfig.Username, mysqlConfig.Password, mysqlConfig.Host, mysqlConfig.Port, mysqlConfig.Database),
			})

			if err != nil {
				logger.Panic("failed to create mysql backend: ", err)
				return
			}
		default:
			backend = btree.NewBTreeCache(logger)
		}

		// bootstrap etcd adapter
		adapter := adapter.NewEtcdAdapter(backend, logger)

		ln, err := net.Listen("tcp", net.JoinHostPort(config.Config.Server.Host, config.Config.Server.Port))
		if err != nil {
			panic(err)
		}
		go func() {
			if err := adapter.Serve(context.Background(), ln); err != nil {
				logger.Panic(err)
				return
			}
		}()

		// graceful exit
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-quit:
			err := adapter.Shutdown(context.TODO())
			if err != nil {
				logger.Error("An error occurred while exiting: ", err)
				return
			}
			err = logger.Sync()
			if err != nil {
				logger.Error("An error occurred while exiting: ", err)
				return
			}
			logger.Info("See you next time!")
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
