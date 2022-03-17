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
	p, _ := filepath.Abs("../etcd-adapter")
	c := exec.Command(p, "-c", config)
	err := c.Start()
	if err != nil {
		return nil, err
	}

	// Determine if the ETCD Adapter started successfully by
	// checking HTTP accessibility
	// The default will retry 5 times, HTTP timeout (1s) time with
	// waiting time (1s) not more than 10 seconds
	var remainRetries = 5
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
