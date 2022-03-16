package rdms_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/api7/etcd-adapter/test/e2e/base"
)

var (
	configFile, _      = filepath.Abs("../../testdata/config/mysql.yaml")
	httpClient         = resty.New().SetTimeout(time.Second)
	etcdClient         *clientv3.Client
	etcdAdapterProcess *exec.Cmd
)

var _ = Describe("MySQL datasource", func() {
	It("Run ETCD Adapter", func() {
		var err error
		etcdAdapterProcess, err = base.RunETCDAdapter(configFile)
		Expect(err).To(BeNil())
		for {
			resp, err := httpClient.R().Get("http://127.0.0.1:12379")
			if err != nil {
				continue
			}
			if resp.StatusCode() == 404 {
				break
			}
			time.Sleep(time.Second)
		}
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
