package server_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/api7/etcd-adapter/test/e2e/base"
)

var _ = Describe("HTTP", func() {
	It("Get version", func() {
		resp, err := base.GetHTTPClient().R().Get("http://" + base.DefaultAddress + "/version")
		Expect(err).To(BeNil())

		json := gjson.ParseBytes(resp.Body())
		Expect(json.Get("etcdserver").String()).To(Equal("3.5.0-pre"))
		Expect(json.Get("etcdcluster").String()).To(Equal("3.5.0"))
	})
})
