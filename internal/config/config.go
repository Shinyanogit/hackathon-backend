package config

import "github.com/caarlos0/env/v9"

type Config struct {
	Port                   string `env:"PORT" envDefault:"8080"`
	DBUser                 string `env:"DB_USER,required"`
	DBPassword             string `env:"DB_PASSWORD,required"`
	DBHost                 string `env:"DB_HOST"` // e.g. tcp(host:3306) or unix(/cloudsql/instance) or empty when using INSTANCE_CONNECTION_NAME
	DBName                 string `env:"DB_NAME,required"`
	DBPort                 string `env:"DB_PORT" envDefault:"3306"`
	InstanceConnectionName string `env:"INSTANCE_CONNECTION_NAME"` // for Cloud SQL unix socket: project:region:instance
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	if cfg.DBHost == "" && cfg.InstanceConnectionName == "" {
		return nil, fmt.Errorf("either DB_HOST or INSTANCE_CONNECTION_NAME must be set")
	}
	return &cfg, nil
}
