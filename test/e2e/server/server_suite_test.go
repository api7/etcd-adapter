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
package server_test

import (
	"os/exec"
	"path/filepath"
	"testing"

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
	Expect(err).To(BeNil())
})

var _ = AfterSuite(func() {
	err := etcdAdapterProcess.Process.Kill()
	Expect(err).To(BeNil())
})
