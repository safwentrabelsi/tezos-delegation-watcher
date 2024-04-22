package config

import "github.com/go-playground/validator/v10"

// ConfigYAML is a transitional struct that contains all the configuration settings, mirroring the structure of the Config struct.
type configYAML struct {
	Server *serverConfigYAML `yaml:"server"`
	Log    *logConfigYAML    `yaml:"log"`
	Tzkt   *tzktConfigYAML   `yaml:"tzkt"`
	DB     *dbConfigYAML     `yaml:"db"`
	Poller *pollerConfigYAML `yaml:"poller"`
}

// dbConfigYAML is a transitional struct used for unmarshaling the database configuration from YAML.

type serverConfigYAML struct {
	Host         string `yaml:"host" validate:"required"`
	Port         int    `yaml:"port" validate:"required,gte=1024,lte=49151"`
	MetricsPort  int    `yaml:"metricsPort" validate:"required,gte=1024,lte=49151"`
	MinValidYear int    `yaml:"minValidYear" validate:"required,gte=2018"`
}

type tzktConfigYAML struct {
	Timeout       int    `yaml:"timeout" validate:"required,gte=0"`
	URL           string `yaml:"url" validate:"required,url"`
	RetryAttempts int    `yaml:"retryAttempts" validate:"required,gte=0"`
}

type dbConfigYAML struct {
	User     string `yaml:"user" validate:"required"`
	DBName   string `yaml:"dbname" validate:"required"`
	Password string `yaml:"password" validate:"required"`
	Host     string `yaml:"host" validate:"required"`
	Port     int    `yaml:"port" validate:"required,gte=1024,lte=49151"`
}

// logConfigYAML is a transitional struct used for unmarshaling the log configuration from YAML.
type logConfigYAML struct {
	Level string `yaml:"level" validate:"required"`
}

// pollerConfigYAML is a transitional struct used for unmarshaling the Poller configuration from YAML.
type pollerConfigYAML struct {
	StartLevel    uint64 `yaml:"startLevel" validate:"required"`
	RetryAttempts int    `yaml:"retryAttempts" validate:"required,gte=0"`
	FetchOld      bool   `yaml:"fetchOld"`
}

var validate *validator.Validate

func init() {
	validate = validator.New()
}
