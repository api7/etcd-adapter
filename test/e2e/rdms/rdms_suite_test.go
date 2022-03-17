package rdms_test

import (
	"os/exec"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	etcdClient         *clientv3.Client
	etcdAdapterProcess *exec.Cmd
)

func TestRDMS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RDMS Suite")
}
