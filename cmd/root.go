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
	"strings"
	"syscall"

	"github.com/api7/gopkg/pkg/log"
	"github.com/k3s-io/kine/pkg/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/api7/etcd-adapter/pkg/adapter"
	"github.com/api7/etcd-adapter/pkg/backends/btree"
	"github.com/api7/etcd-adapter/pkg/backends/fdb"
	"github.com/api7/etcd-adapter/pkg/backends/mysql"
	"github.com/api7/etcd-adapter/pkg/config"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "etcd-adapter",
	Short: "The bridge between etcd protocol and other storage backends.",
	Run: func(cmd *cobra.Command, args []string) {
		// initialize configuration
		err := config.Init(configFile)
		if err != nil {
			dief("failed to initialize configuration: %s", err)
		}

		// initialize log
		log.DefaultLogger, err = log.NewLogger(
			log.WithSkipFrames(3),
			log.WithLogLevel(config.Config.Log.Level),
		)
		if err != nil {
			dief("failed to initialize logging: %s", err)
		}

		// initialize backend
		log.Infow("using backends type", zap.String("datasources", string(config.Config.DataSource.Type)))
		var backend server.Backend
		switch config.Config.DataSource.Type {
		case config.Mysql:
			mysqlConfig := config.Config.DataSource.MySQL
			backend, err = mysql.NewMySQLCache(context.TODO(), &mysql.Options{
				DSN: fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", mysqlConfig.Username, mysqlConfig.Password, mysqlConfig.Host, mysqlConfig.Port, mysqlConfig.Database),
			})

			if err != nil {
				dief("failed to create mysql backend, err: %s", err)
			}
		case config.BTree:
			backend = btree.NewBTreeCache()
		case config.FDB:
			fdbConfig := config.Config.DataSource.FDB
			backend, err = fdb.NewFDBCache(context.TODO(), &fdb.Options{
				ClusterFile: fdbConfig.ClusterFile,
				Directory:   strings.Split(fdbConfig.Directory, "/"),
			})
			if err != nil {
				dief("failed to create FDB backend, err: %s", err)
			}
		default:
			dief("does not support backends from %s", config.Config.DataSource.Type)
		}

		// bootstrap etcd adapter
		adapter := adapter.NewEtcdAdapter(&adapter.AdapterOptions{
			Backend: backend,
		})

		log.Info("configuring listeners at ", config.Config.Server.Host, ":", config.Config.Server.Port, "")
		ln, err := net.Listen("tcp", net.JoinHostPort(config.Config.Server.Host, config.Config.Server.Port))
		if err != nil {
			dief("failed create listenners, err:", err)

		}
		go func() {
			log.Info("start etcd-adapter server")
			if err := adapter.Serve(context.Background(), ln); err != nil {
				dief("failed to start etcd-adapter server, err:", err)
			}
		}()

		// graceful exit
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		<-quit
		err = adapter.Shutdown(context.TODO())
		if err != nil {
			log.Error("An error occurred while exiting: ", err)
			return
		}
		err = log.DefaultLogger.Sync()
		if err != nil {
			log.Error("An error occurred while exiting: ", err)
			return
		}
		log.Info("etcd-adapter exit")
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

func dief(template string, args ...interface{}) {
	if !strings.HasSuffix(template, "\n") {
		template += "\n"
	}
	fmt.Fprintf(os.Stderr, template, args...)
	os.Exit(1)
}
