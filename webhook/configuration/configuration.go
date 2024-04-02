package configuration

import (
	"time"

	"github.com/caarlos0/env/v8"
	log "github.com/sirupsen/logrus"
)

// Configuration struct for configuration environment variables
type Configuration struct {
	ApiKey             string        `env:"ABION_API_KEY"`
	Debug              bool          `env:"ABION_DEBUG" default:"false"`
	LogFormat          string        `env:"LOG_FORMAT" default:"text"`
	DryRun             bool          `env:"DRY_RUN" default:"false"`
	ServerHost         string        `env:"SERVER_HOST" envDefault:"localhost"`
	ServerPort         int           `env:"SERVER_PORT" envDefault:"8888"`
	ServerReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"0"`
	ServerWriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"0"`
}

// Init sets up configuration by reading environmental variables
func Init() Configuration {
	cfg := Configuration{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Error reading configuration from environment: %v", err)
	}
	if cfg.ApiKey == "" {
		panic("ABION_API_KEY must be specified")
	}

	return cfg
}
