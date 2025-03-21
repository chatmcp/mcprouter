package util

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// InitConfigWithFile will read config from file
func InitConfigWithFile(filename string) error {
	viper.SetConfigFile(filename)

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("Config file changed: %s\n", e.Name)
	})

	return nil
}
