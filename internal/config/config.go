package config

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/api7/etcd-adapter/internal/utils"
)

var (
	// the DSN for database
	DSN string

	// etcd Server and TLS configuration
	Server server
)

// Init load and unmarshal config file
func Init(configFile string) error {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		utils.GetLogger().Error("Config file load failed", zap.Error(err))
		return err
	}
	utils.GetLogger().Info("Config file load successful", zap.String("path", viper.ConfigFileUsed()))

	err = unmarshal()
	if err != nil {
		utils.GetLogger().Error("Config file unmarshal failed", zap.Error(err))
		return err
	}

	return nil
}

func unmarshal() error {
	err := viper.UnmarshalKey("server", &Server)
	if err != nil {
		return err
	}

	err = viper.UnmarshalKey("dsn", &DSN)
	if err != nil {
		return err
	}

	return nil
}
