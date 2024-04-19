package config

// ConfigYAML is a transitional struct that contains all the configuration settings, mirroring the structure of the Config struct.
type configYAML struct {
	Server *serverConfigYAML `yaml:"server"`
	Log    *logConfigYAML    `yaml:"log"`
	Tzkt   *tzktConfigYAML   `yaml:"tzkt"`
	DB     *dbConfigYAML     `yaml:"db"`
	Poller *pollerConfigYAML `yaml:"poller"`
}

// serverConfigYAML is a transitional struct used for unmarshaling the server configuration from YAML.
type serverConfigYAML struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	MetricsPort  int    `yaml:"metricsPort"`
	MinValidYear int    `yaml:"minValidYear"`
}

// logConfigYAML is a transitional struct used for unmarshaling the log configuration from YAML.
type logConfigYAML struct {
	Level string `yaml:"level"`
}

type tzktConfigYAML struct {
	Timeout int    `yaml:"timeout"`
	URL     string `yaml:"url"`
}

// dbConfigYAML is a transitional struct used for unmarshaling the database configuration from YAML.
type dbConfigYAML struct {
	User     string `yaml:"user"`
	DBName   string `yaml:"dbname"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
}

// pollerConfigYAML is a transitional struct used for unmarshaling the Poller configuration from YAML.
type pollerConfigYAML struct {
	StartLevel uint64 `yaml:"startLevel"`
}
