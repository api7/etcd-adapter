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
package datasource_test

import (
	"context"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/api7/etcd-adapter/test/e2e/base"
)

var (
	mysqlConfigFile, _ = filepath.Abs("../../testdata/config/mysql.yaml")
)

var _ = Describe("MySQL datasource", func() {
	It("Run ETCD Adapter", func() {
		var err error
		etcdAdapterProcess, err = base.RunETCDAdapter(mysqlConfigFile)
		Expect(err).To(BeNil())
	})

	It("Connecting to ETCD Adapter", func() {
		var err error
		etcdClient, err = base.NewETCDClient()
		Expect(err).To(BeNil())
	})

	It("Get from ETCD Adapter", func() {
		resp, err := etcdClient.KV.Get(context.TODO(), "/", clientv3.WithPrefix())
		Expect(err).To(BeNil())
		Expect(len(resp.Kvs)).To(BeNumerically(">", 0))
	})

	It("Stop ETCD Adapter", func() {
		err := etcdClient.Close()
		Expect(err).To(BeNil())

		err = etcdAdapterProcess.Process.Kill()
		Expect(err).To(BeNil())
	})
})
