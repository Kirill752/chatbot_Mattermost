package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	MM_TOKEN    string `yaml:"MM_TOKEN" env-required:"true"`
	MM_USERNAME string `yaml:"MM_USERNAME" env-required:"true"`
	MM_TEAM     string `yaml:"MM_TEAM" env-required:"true"`
	MM_CHANNEL  string `yaml:"MM_CHANNEL" env-required:"true"`
	MM_SERVER   string `yaml:"MM_SERVER" env-required:"true"`
}

func MustLoad(configPath string) *Config {
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file %s does not exist", configPath)
	}
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("can not read config file: %s", configPath)
	}
	return &cfg
}
