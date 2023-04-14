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
package apisix_test

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/api7/etcd-adapter/test/e2e/base"
)

var (
	serverConfigFile, _ = filepath.Abs("../../testdata/config/server.yaml")
	etcdAdapterProcess  *exec.Cmd
	expect              *httpexpect.Expect
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = BeforeSuite(func() {
	var err error
	etcdAdapterProcess, err = base.RunETCDAdapter(serverConfigFile)
	expect = httpexpect.New(GinkgoT(), "http://127.0.0.1:9080")

	Expect(err).To(BeNil())

	cmd := exec.Command("docker-compose", "-f", "../../testdata/apisix-adapter.docker-compose.yaml", "up", "-d")
	err = cmd.Run()
	Expect(err).NotTo(HaveOccurred())
	httpexpect.New(GinkgoT(), "http://127.0.0.1:9080").GET("").
		WithRetryPolicy(httpexpect.RetryAllErrors).
		WithRetryDelay(200*time.Millisecond, 10*time.Second).
		WithMaxRetries(10).
		Expect().Status(404)
})

var _ = AfterSuite(func() {
	cmd := exec.Command("docker-compose", "-f", "../../testdata/apisix-adapter.docker-compose.yaml", "down")
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
	err = etcdAdapterProcess.Process.Kill()
	Expect(err).To(BeNil())
})
