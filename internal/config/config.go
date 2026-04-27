package config

import (
	"context"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Environment string `env:"ENVIRONMENT,required"`
	HttpHost    string `env:"HTTP_HOST,required"`
	HttpPort    string `env:"HTTP_PORT,required"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.ProcessWith(context.TODO(), &envconfig.Config{Target: &cfg}); err != nil {
		return nil, err
	}

	return &cfg, nil
}
