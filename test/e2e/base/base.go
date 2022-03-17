package base

import (
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	DefaultAddress = "127.0.0.1:12379"
)

var (
	httpClient = resty.New().SetTimeout(time.Second)
)

// GetHTTPClient returns the default HTTP client
// timeout: 1 second
func GetHTTPClient() *resty.Client {
	return httpClient
}
