package config

type datasourceType string

const (
	Mysql  datasourceType = "mysql"
	BTree  datasourceType = "btree"
	SQLite datasourceType = "sqlite"
)

type server struct {
	Host string    `mapstructure:"host"`
	Port string    `mapstructure:"port"`
	TLS  serverTLS `mapstructure:"tls"`
}

type serverTLS struct {
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

type log struct {
	Level string `mapstructure:"level"`
}

type mysqlConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

type datasource struct {
	Type  datasourceType `mapstructure:"type"`
	MySQL mysqlConfig    `mapstructure:"mysql"`
}

type config struct {
	Server     server     `mapstructure:"server"`
	Log        log        `mapstructure:"log"`
	DataSource datasource `mapstructure:"datasource"`
}
