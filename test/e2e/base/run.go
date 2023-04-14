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
package base

import (
	"errors"
	"net/http"
	"os/exec"
	"path/filepath"
	"time"
)

// RunETCDAdapter start an etcd adapter instance
func RunETCDAdapter(config string) (*exec.Cmd, error) {
	p, _ := filepath.Abs("../../../etcd-adapter")
	c := exec.Command(p, "-c", config)
	if err := c.Start(); err != nil {
		return nil, err
	}
	// Determine if the ETCD Adapter started successfully by
	// checking HTTP accessibility
	// The default will retry 5 times, HTTP timeout (1s) time with
	// waiting time (1s) not more than 10 seconds
	var remainRetries = 1
	for {
		if remainRetries == 0 {
			return nil, errors.New("exceeds the number of run check retries")
		}
		client := &http.Client{Timeout: time.Second}
		resp, err := client.Get("http://" + DefaultAddress)

		if err != nil {
			continue
		}
		if resp.StatusCode == 404 {
			return c, nil
		}
		remainRetries--
		time.Sleep(time.Second)
	}
}
