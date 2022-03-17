package base

import (
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// NewETCDClient create an ETCD client for testing.
func NewETCDClient() (*clientv3.Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{DefaultAddress},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return cli, nil
}
