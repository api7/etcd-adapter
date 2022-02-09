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

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/api7/etcd-adapter/internal/adapter"
	"github.com/api7/etcd-adapter/internal/backends/mysql"
)

var rootCmd = &cobra.Command{
	Use:   "etcd-adapter",
	Short: "The bridge between etcd protocol and other storage backends.",
	Run: func(cmd *cobra.Command, args []string) {
		opts := &adapter.AdapterOptions{
			Logger:  zap.NewExample(),
			Backend: adapter.BackendMySQL,
			MySQLOptions: &mysql.Options{
				DSN: "root@tcp(127.0.0.1:3306)/apisix",
			},
		}
		a := adapter.NewEtcdAdapter(opts)

		ln, err := net.Listen("tcp", "127.0.0.1:12379")
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
			if err != nil {
				opts.Logger.Error("An error occurred while exiting.", zap.Error(err))
				return
			}
			opts.Logger.Info("See you next time!")
		}
	},
}

// Execute bootstrap root command.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
