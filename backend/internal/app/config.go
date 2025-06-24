package app

import (
	"log"

	"github.com/spf13/viper"
)

// config is the struct for config file
type config struct {
	App AppConfig `yaml:"app"`
	DB  DBConfig  `yaml:"db"`
}

type AppConfig struct {
	Host      string `yaml:"host"`
	Port      int32  `yaml:"port"`
	RunMode   string `yaml:"runMode"`
	JWTSecret string `yaml:"jwtSecret"`
}

// DBConfig is the struct for database config
type DBConfig struct {
	Type string `yaml:"type"`
	DNS  string `yaml:"dns"`
}

// configInstance is the singleton instance of the config
var configInstance *config

// Config returns a singleton instance of the config
func Config() *config {
	if configInstance == nil {
		viper.SetConfigFile("config/config.yaml")

		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("error reading config file, %v", err)
		}

		if err := viper.Unmarshal(&configInstance); err != nil {
			log.Fatalf("unable to decode config into struct, %v", err)
		}
	}

	return configInstance
}
