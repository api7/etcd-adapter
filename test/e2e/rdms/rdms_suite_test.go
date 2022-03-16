package rdms_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRDMS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RDMS Suite")
}
