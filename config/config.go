package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config contains all top-level configuration settings for the application, accessible for reading.
type Config struct {
	Server *ServerConfig
	Log    *LogConfig
	Tzkt   *TzktConfig
	DB     *DBConfig
	Poller *PollerConfig
}

// ServerConfig contains configuration details for the server, with fields unexported for encapsulation.
type ServerConfig struct {
	host         string
	port         int
	metricsPort  int
	minValidYear int
}

// LogConfig contains configuration settings for logging.
type LogConfig struct {
	level string
}

// TzktConfig contains configuration details for interacting with the Tzkt API.
type TzktConfig struct {
	timeout       int
	url           string
	retryAttempts int
}

// DBConfig contains database connection settings with sensitive details unexported.
type DBConfig struct {
	user     string
	dbname   string
	password string
	host     string
	port     int
}

// PollerConfig contains poller settings.
type PollerConfig struct {
	startLevel    uint64
	retryAttempts int
}

var (
	cfg     *Config
	once    sync.Once
	loadErr error
)

// LoadConfig reads configuration from the given file.
func LoadConfig(configFile string) (*Config, error) {
	var loadErr error
	once.Do(func() {
		cfg = &Config{}

		absPath, err := filepath.Abs(configFile)
		if err != nil {
			loadErr = fmt.Errorf("Error finding absolute path for the configuration file: %v", err)
			return
		}

		yamlFile, err := os.ReadFile(absPath)
		if err != nil {
			loadErr = fmt.Errorf("Error reading YAML file: %v", err)
			return
		}

		configYAML := configYAML{}
		err = yaml.Unmarshal(yamlFile, &configYAML)
		if err != nil {
			loadErr = fmt.Errorf("Error parsing YAML file: %v", err)
			return
		}

		// Perform validation
		if err := validate.Struct(configYAML); err != nil {
			loadErr = fmt.Errorf("validation error: %v", err)
			return
		}
		cfg.Server = &ServerConfig{
			host:         configYAML.Server.Host,
			port:         configYAML.Server.Port,
			metricsPort:  configYAML.Server.MetricsPort,
			minValidYear: configYAML.Server.MinValidYear,
		}
		cfg.Log = &LogConfig{
			level: configYAML.Log.Level,
		}
		cfg.Tzkt = &TzktConfig{
			timeout:       configYAML.Tzkt.Timeout,
			url:           configYAML.Tzkt.URL,
			retryAttempts: configYAML.Tzkt.RetryAttempts,
		}
		cfg.DB = &DBConfig{
			user:     configYAML.DB.User,
			dbname:   configYAML.DB.DBName,
			password: configYAML.DB.Password,
			host:     configYAML.DB.Host,
			port:     configYAML.DB.Port,
		}
		cfg.Poller = &PollerConfig{
			startLevel:    configYAML.Poller.StartLevel,
			retryAttempts: configYAML.Tzkt.RetryAttempts,
		}
	})

	return cfg, loadErr
}

// GetHost returns the host configuration from the ServerConfig.
func (s *ServerConfig) GetHost() string {
	return s.host
}

// GetPort returns the port configuration from the ServerConfig.
func (s *ServerConfig) GetPort() int {
	return s.port
}

// GetMetricsPort returns the metrics port configuration from the ServerConfig.
func (s *ServerConfig) GetMetricsPort() int {
	return s.metricsPort
}

// GetMinValidYear returns the minnimum valid year from the ServerConfig.
func (s *ServerConfig) GetMinValidYear() int {
	return s.minValidYear
}

// GetLevel returns the host configuration from LogConfig.
func (l *LogConfig) GetLevel() string {
	return l.level
}

// GetTimeout returns the timeout configuration from the TzktConfig.
func (t *TzktConfig) GetTimeout() int {
	return t.timeout
}

// GetURL returns the url configuration from the TzktConfig.
func (t *TzktConfig) GetURL() string {
	return t.url
}

// GetRetryAttempts returns the maximum retry attempts from the TzktConfig.
func (t *TzktConfig) GetRetryAttempts() int {
	return t.retryAttempts
}

// GetStartLevel returns the start level configuration from the pollerConfig.
func (p *PollerConfig) GetStartLevel() uint64 {
	return p.startLevel
}

// GetRetryAttempts returns the maximum retry attempts from the pollerConfig.
func (p *PollerConfig) GetRetryAttempts() int {
	return p.retryAttempts
}

// GetUser returns the user configuration from the DBConfig.
func (d *DBConfig) GetUser() string {
	return d.user
}

// GetDbname returns the name configuration from the DBConfig.
func (d *DBConfig) GetDbname() string {
	return d.dbname
}

// GetPassword returns the password configuration from the DBConfig.
func (d *DBConfig) GetPassword() string {
	return d.password
}

// GetHost returns the host configuration from the DBConfig.
func (d *DBConfig) GetHost() string {
	return d.host
}

// GetPort returns the port configuration from the DBConfig.
func (d *DBConfig) GetPort() int {
	return d.port
}

// GetPostgresqlDSN constructs a PostgreSQL DSN from the DBConfig.
func (d *DBConfig) GetPostgresqlDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", d.user, d.password, d.host, d.port, d.dbname)
}

// GetListenAddress constructs the listenning address from the ServerConfig.
func (s *ServerConfig) GetListenAddress() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}
