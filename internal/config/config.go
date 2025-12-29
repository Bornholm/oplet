package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/pkg/errors"
)

type Config struct {
	Logger  Logger  `envPrefix:"LOGGER_"`
	HTTP    HTTP    `envPrefix:"HTTP_"`
	Storage Storage `envPrefix:"STORAGE_"`
	Seed    Seed    `envPrefix:"SEED_"`
	Runner  Runner  `envPrefix:"RUNNER_"`
}

func Parse() (*Config, error) {
	conf, err := env.ParseAsWithOptions[Config](env.Options{
		Prefix: "OPLET_",
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &conf, nil
}
