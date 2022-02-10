package config

type server struct {
	Host string
	Port string
	TLS  serverTLS `mapstructure:"tls"`
}

type serverTLS struct {
	Cert string
	Key  string
}
