// Package config to env
package config

import "github.com/caarlos0/env/v6"

// Config struct to postgres config env
type Config struct {
	DB             string `env:"DB" envDefault:"postgres"`
	User           string `env:"POSTGRES_USER" envDefault:"egormelnikov"`
	Password       string `env:"POSTGRES_PASSWORD" envDefault:"54236305"`
	Host           string `env:"POSTGRES_HOST" envDefault:"postgres"`
	PortPostgres   string `env:"POSTGRES_PORT" envDefault:"5432"`
	DBNamePostgres string `env:"POSTGRES_DB" envDefault:"egormelnikov"`
	DBURL          string `env:"DBURL" envDefault:""`
	PricePort      string `env:"PRICE_PORT" envDefault:"8089"`
}

// New contract config
func New() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
