package config

type server struct {
	Host string    `mapstructure:"host"`
	Port string    `mapstructure:"port"`
	TLS  serverTLS `mapstructure:"tls"`
}

type serverTLS struct {
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

type mysqlConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

type datasource struct {
	Type  string      `mapstructure:"type"`
	MySQL mysqlConfig `mapstructure:"mysql"`
}

type config struct {
	Server     server     `mapstructure:"server"`
	DataSource datasource `mapstructure:"datasource"`
}
