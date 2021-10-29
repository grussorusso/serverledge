package config

import (
	"log"

	"github.com/spf13/viper"
)

//Get returns the configured value for a given key or the specified default.
func Get(key string, defaultValue interface{}) interface{} {
	if viper.IsSet(key) {
		return viper.Get(key)
	} else {
		return defaultValue
	}
}

func GetInt(key string, defaultValue int) int {
	if viper.IsSet(key) {
		return viper.GetInt(key)
	} else {
		return defaultValue
	}
}

func GetString(key string, defaultValue string) string {
	if viper.IsSet(key) {
		return viper.GetString(key)
	} else {
		return defaultValue
	}
}

func GetBool(key string, defaultValue bool) bool {
	if viper.IsSet(key) {
		return viper.GetBool(key)
	} else {
		return defaultValue
	}
}

//ReadConfiguration reads a configuration file stored in one of the predefined paths.
func ReadConfiguration() {
	viper.SetConfigName("serverledge-conf") // name of config file (without extension)

	// paths where the config file can be placed
	viper.AddConfigPath("/etc/serverledge/")
	viper.AddConfigPath("$HOME/")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("No configuration file parsed.")
		} else {
			log.Printf("Config file parsing failed!")
		}
	}
}
