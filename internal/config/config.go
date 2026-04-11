package config

import (
	"context"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	HttpHost string `env:"HTTP_HOST"`
	HttpPort string `env:"HTTP_PORT, default=8090"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.ProcessWith(context.TODO(), &envconfig.Config{Target: &cfg}); err != nil {
		return nil, err
	}

	return &cfg, nil
}
