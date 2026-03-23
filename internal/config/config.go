package config

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	App AppConfig
	DB  DBConfig
	JWT JWTConfig
}

type AppConfig struct {
	Port string `envconfig:"APP_PORT" default:"8080"`
	Env  string `envconfig:"APP_ENV" default:"development"`
}

type DBConfig struct {
	Host     string `envconfig:"DB_HOST" required:"true"`
	Port     string `envconfig:"DB_PORT" default:"5432"`
	User     string `envconfig:"DB_USER" required:"true"`
	Password string `envconfig:"DB_PASSWORD" required:"true"`
	Name     string `envconfig:"DB_NAME" requured:"true"`
	SSLMode  string `envconfig:"DB_SSLMODE" default:"disable"`
}

type JWTConfig struct {
	Secret     string        `envconfig:"JWT_SECRET" required:"true"`
	AccessTTL  time.Duration `envconfig:"JWT_ACCESS_TTL" default:"15m"`
	RefreshTTL time.Duration `envconfig:"JWT_REFRESH_TTL" default:"168h"`
}

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env filem reading from environment")
	}

	var cfg Config

	for _, s := range []struct {
		prefix string
		spec   interface{}
	}{
		{"APP", &cfg.App},
		{"DB", &cfg.DB},
		{"JWT", &cfg.JWT},
	} {
		if err := envconfig.Process(s.prefix, s.spec); err != nil {
			log.Fatalf("config error: %s", err)
		}
	}
	return &cfg
}
