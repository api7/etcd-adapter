package base

import (
	"os/exec"
	"path/filepath"
)

// RunETCDAdapter start an etcd adapter instance
func RunETCDAdapter(config string) (*exec.Cmd, error) {
	p, _ := filepath.Abs("../etcd-adapter")
	c := exec.Command(p, "-c", config)
	err := c.Start()
	if err != nil {
		return nil, err
	}

	return c, nil
}
