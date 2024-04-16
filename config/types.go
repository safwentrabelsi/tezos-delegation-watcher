package config

// ConfigYAML is a transitional struct that contains all the configuration settings, mirroring the structure of the Config struct.
type configYAML struct {
	Server     serverConfigYAML     `yaml:"server"`
	Log        logConfigYAML        `yaml:"log"`
	Validation validationConfigYAML `yaml:"validation"`
	Tzkt       tzktConfigYAML       `yaml:"tzkt"`
	DB         dbConfigYAML         `yaml:"db"`
	Metrics    metricsConfigYAML    `yaml:"metrics"`
}

// serverConfigYAML is a transitional struct used for unmarshaling the server configuration from YAML.
type serverConfigYAML struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// logConfigYAML is a transitional struct used for unmarshaling the log configuration from YAML.
type logConfigYAML struct {
	Level string `yaml:"level"`
}

// validationConfigYAML is a transitional struct used for unmarshaling the validation configuration from YAML.
type validationConfigYAML struct {
	StartYear int `yaml:"startYear"`
}

// tzktConfigYAML is a transitional struct used for unmarshaling the Tzkt configuration from YAML.
type tzktConfigYAML struct {
	Timeout    int    `yaml:"timeout"`
	URL        string `yaml:"url"`
	StartLevel uint64 `yaml:"startLevel"`
}

// dbConfigYAML is a transitional struct used for unmarshaling the database configuration from YAML.
type dbConfigYAML struct {
	User     string `yaml:"user"`
	DBName   string `yaml:"dbname"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
}

// metricsConfigYAML is a transitional struct used for unmarshaling the metrics configuration from YAML.
type metricsConfigYAML struct {
	Port int `yaml:"port"`
}
