package server_test

import (
	"github.com/gavv/httpexpect/v2"
	. "github.com/onsi/ginkgo/v2"

	"github.com/api7/etcd-adapter/test/e2e/base"
)

var _ = Describe("HTTP", func() {
	e := httpexpect.New(GinkgoT(), "http://"+base.DefaultAddress)

	It("Get version", func() {
		resp := e.GET("/version").Expect()
		resp.Status(200)
		resp.JSON().
			Object().
			ValueEqual("etcdserver", "3.5.0").
			ValueEqual("etcdcluster", "3.5.0")
	})
})
