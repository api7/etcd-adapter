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
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = BeforeSuite(func() {
	cmd := exec.Command("docker compose", "-f", "../../testdata/apisix-adapter.docker-compose.yaml", "up", "-d")
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
	httpexpect.New(GinkgoT(), "http://127.0.0.1:9080").GET("").
		WithRetryPolicy(httpexpect.RetryAllErrors).
		WithRetryDelay(200*time.Millisecond, 2*time.Second).
		WithMaxRetries(6).
		Expect().Status(404)

	httpexpect.New(GinkgoT(), "http://127.0.0.1:12379").GET("").
		WithRetryPolicy(httpexpect.RetryAllErrors).
		WithRetryDelay(200*time.Millisecond, 2*time.Second).
		WithMaxRetries(6).
		Expect().Status(404)
})

var _ = AfterSuite(func() {
	cmd := exec.Command("docker-compose", "-f", "../../testdata/apisix-adapter.docker-compose.yaml", "down")
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
})
