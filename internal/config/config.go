// Package config to env
package config

import "github.com/caarlos0/env/v6"

// Config struct to postgres config env
type Config struct {
	DB             string `env:"DB" envDefault:"postgres"`
	User           string `env:"USER" envDefault:"egormelnikov"`
	Password       string `env:"PASSWORD" envDefault:"54236305"`
	Host           string `env:"HOST" envDefault:"postgres"`
	PortPostgres   int    `env:"PORT" envDefault:"5432"`
	DBNamePostgres string `env:"DBNAMEPOSTGRES" envDefault:"egormelnikovdb"`
	DBURL          string `env:"DBURL" envDefault:""`
}

// New contract config
func New() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
