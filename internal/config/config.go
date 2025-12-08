package config

import "github.com/caarlos0/env/v9"

type Config struct {
	Port       string `env:"PORT" envDefault:"8080"`
	DBUser     string `env:"DB_USER,required"`
	DBPassword string `env:"DB_PASSWORD,required"`
	DBHost     string `env:"DB_HOST,required"` // e.g. tcp(host:3306) or unix(/cloudsql/instance)
	DBName     string `env:"DB_NAME,required"`
	DBPort     string `env:"DB_PORT" envDefault:"3306"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
