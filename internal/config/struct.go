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

type pgsqlConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

type sqliteConfig struct {
	File string `mapstructure:"file"`
}

type datasource struct {
	Type   string       `mapstructure:"type"`
	MySQL  mysqlConfig  `mapstructure:"mysql"`
	PgSQL  pgsqlConfig  `mapstructure:"pgsql"`
	SQLite sqliteConfig `mapstructure:"sqlite"`
}

type config struct {
	Server     server     `mapstructure:"server"`
	Log        log        `mapstructure:"log"`
	DataSource datasource `mapstructure:"datasource"`
}
