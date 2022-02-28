package config

import (
	"strings"

	loggkg "github.com/api7/gopkg/pkg/log"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	// the Config for etcd adapter
	Config config
)

// Init load and unmarshal config file
func Init(configFile string, logger *loggkg.Logger) error {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
	viper.AddConfigPath("config")

	// read configuration file
	viper.SetEnvPrefix("EA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		logger.Errorw("Config file load failed", zap.Error(err))
		return err
	}
	logger.Infow("Config file load successful", zap.String("path", viper.ConfigFileUsed()))

	// parse configuration
	err = viper.Unmarshal(&Config)
	if err != nil {
		logger.Errorw("Config file unmarshal failed", zap.Error(err))
		return err
	}

	return nil
}
