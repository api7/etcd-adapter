package server_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/api7/etcd-adapter/test/e2e/base"
)

var (
	serverConfigFile, _ = filepath.Abs("../../testdata/config/server.yaml")
	etcdAdapterProcess  *exec.Cmd
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
