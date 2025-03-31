package config

import (
	"errors"

	"github.com/spf13/viper"

	"github.com/agntcy/dir/cli/util/dir"
)

func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath(dir.GetAppDir())

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return nil
		} else {
			return err
		}
	}

	return nil
}
