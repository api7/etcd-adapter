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
	"time"

	"github.com/gavv/httpexpect/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("APISIX", func() {
	e := httpexpect.New(GinkgoT(), "http://127.0.0.1:9080")
	//admin := httpexpect.New(GinkgoT(), "http://127.0.0.1:9180")

	It("Get version", func() {
		/*
			admin.PUT("/apisix/admin/routes/1").
				WithHeader("X-API-KEY", "edd1c9f034335f136f87ad84b625c8f1").
				WithBytes([]byte(`{"update_time":1681107963,"create_time":1681107963,"status":1,"upstream":{"pass_host":"pass","nodes":{"postman-echo.com:80":1},"scheme":"http","type":"roundrobin","hash_on":"vars"},"uri":"/get","id":"1","priority":0}`)).
				Expect().Status(201)
		*/
		time.Sleep(30 * time.Minute)
		cmd := exec.Command("etcdctl", "--endpoints=127.0.0.1:12379", "put", "/apisix/routes/1", `{"update_time":1681107963,"create_time":1681107963,"status":1,"upstream":{"pass_host":"pass","nodes":{"postman-echo.com:80":1},"scheme":"http","type":"roundrobin","hash_on":"vars"},"uri":"/get","id":"1","priority":0}`)
		err := cmd.Run()
		Expect(err).To(BeNil())
		e.GET("/get").Expect().Status(200)
	})
})
