package config

import (
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel      string        `yaml:"log_level"      env:"LOG_LEVEL"       env-default:"DEBUG"`
	Address       string        `yaml:"search_address" env:"SEARCH_ADDRESS"  env-default:"localhost:83"`
	DBAddress     string        `yaml:"db_address"     env:"DB_ADDRESS"      env-default:"localhost:5432"`
	WordsAddress  string        `yaml:"words_address"  env:"WORDS_ADDRESS"   env-default:"localhost:81"`
	IndexTTL      time.Duration `yaml:"index_ttl"      env:"INDEX_TTL"       env-default:"5m"`
	BrokerAddress string        `yaml:"broker_address" env:"BROKER_ADDRESS"  env-default:"nats://localhost:4222"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		if err2 := cleanenv.ReadEnv(&cfg); err2 != nil {
			slog.Error("cannot read config", "path", configPath, "error", err)
			os.Exit(1)
		}
	}
	return cfg
}
