package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel   string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address    string        `yaml:"address" env:"FRONTEND_ADDRESS" env-default:":8080"`
	Timeout    time.Duration `yaml:"timeout" env:"FRONTEND_TIMEOUT" env-default:"5s"`
	APIAddress string        `yaml:"api_address" env:"API_ADDRESS" env-default:"http://api:8080"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
