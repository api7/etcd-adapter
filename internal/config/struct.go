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

type mysqlConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

type datasource struct {
	Type  string
	MySQL mysqlConfig `mapstructure:"mysql"`
}

type config struct {
	Server     server
	DataSource datasource `mapstructure:"datasource"`
}
