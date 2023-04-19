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
	"time"

	"github.com/gavv/httpexpect/v2"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("APISIX", func() {
	e := httpexpect.New(GinkgoT(), "http://127.0.0.1:9080")
	admin := httpexpect.New(GinkgoT(), "http://127.0.0.1:9180")

	var (
		createRoute1 = `
{
    "uri": "/headers",
    "upstream": {
        "type": "roundrobin",
        "nodes": {
            "postman-echo.com:80": 1
        }
    }
}`
		updateRoute1 = `
{
    "uri": "/get",
    "upstream": {
        "type": "roundrobin",
        "nodes": {
            "postman-echo.com:80": 1
        }
    }
}`
	)

	It("create route for the apisix, and access it", func() {
		admin.PUT("/apisix/admin/routes/1").
			WithHeader("X-API-KEY", "edd1c9f034335f136f87ad84b625c8f1").
			WithBytes([]byte(createRoute1)).
			Expect().Status(201)

		time.Sleep(2 * time.Second)

		e.GET("/headers").Expect().Status(200)
	})

	It("update route for the apisix, and access it", func() {
		// create route
		admin.PUT("/apisix/admin/routes/1").
			WithHeader("X-API-KEY", "edd1c9f034335f136f87ad84b625c8f1").
			WithBytes([]byte(createRoute1)).
			Expect().Status(201)

		time.Sleep(2 * time.Second)

		e.GET("/headers").Expect().Status(200)

		// update route
		admin.PUT("/apisix/admin/routes/1").
			WithHeader("X-API-KEY", "edd1c9f034335f136f87ad84b625c8f1").
			WithBytes([]byte(updateRoute1)).
			Expect().Status(201)

		time.Sleep(2 * time.Second)

		e.GET("/headers").Expect().Status(404)
		e.GET("/get").Expect().Status(200)
	})
})
