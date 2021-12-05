package config

import (
	"log"

	"github.com/spf13/viper"
)

var DefaultConfigFileName = "serverledge-conf"

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
func ReadConfiguration(fileName string) {
	viper.SetConfigName(DefaultConfigFileName) // default name of config file (without extension)
	if fileName != "" {
		viper.SetConfigName(fileName) //custom name of config file (without extension)
	}

	// paths where the config file can be placed
	viper.AddConfigPath("/etc/serverledge/")
	viper.AddConfigPath("$HOME/")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../serverledge/internal/config")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("No configuration file parsed.")
		} else {
			log.Printf("Config file parsing failed!")
		}
	}
}
