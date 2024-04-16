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
	Server     *ServerConfig
	Log        *LogConfig
	Validation *ValidationConfig
	Tzkt       *TzktConfig
	DB         *DBConfig
	Metrics    *MetricsConfig
}

// ServerConfig contains configuration details for the server, with fields unexported for encapsulation.
type ServerConfig struct {
	host string
	port int
}

// LogConfig contains configuration settings for logging.
type LogConfig struct {
	level string
}

// ValidationConfig contains settings related to data validation.
type ValidationConfig struct {
	startYear int
}

// TzktConfig contains configuration details for interacting with the Tzkt API.
type TzktConfig struct {
	timeout    int
	url        string
	startLevel uint64
	// TODO add api limit
}

// DBConfig contains database connection settings with sensitive details unexported.
type DBConfig struct {
	user     string
	dbname   string
	password string
	host     string
	port     int
}

// MetricsConfig contains settings for exposing metrics on a specific port.
type MetricsConfig struct {
	port int
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

		cfg.Server = &ServerConfig{
			host: configYAML.Server.Host,
			port: configYAML.Server.Port,
		}
		cfg.Log = &LogConfig{
			level: configYAML.Log.Level,
		}
		cfg.Validation = &ValidationConfig{
			startYear: configYAML.Validation.StartYear,
		}
		cfg.Tzkt = &TzktConfig{
			timeout:    configYAML.Tzkt.Timeout,
			url:        configYAML.Tzkt.URL,
			startLevel: configYAML.Tzkt.StartLevel,
		}
		cfg.DB = &DBConfig{
			user:     configYAML.DB.User,
			dbname:   configYAML.DB.DBName,
			password: configYAML.DB.Password,
			host:     configYAML.DB.Host,
			port:     configYAML.DB.Port,
		}
		cfg.Metrics = &MetricsConfig{
			port: configYAML.Metrics.Port,
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

// GetLevel returns the host configuration from LogConfig.
func (l *LogConfig) GetLevel() string {
	return l.level
}

// GetStartYear returns the start year configuration from the ValidationConfig.
func (v *ValidationConfig) GetStartYear() int {
	return v.startYear
}

// GetTimeout returns the timeout configuration from the TzktConfig.
func (t *TzktConfig) GetTimeout() int {
	return t.timeout
}

// GetURL returns the url configuration from the TzktConfig.
func (t *TzktConfig) GetURL() string {
	return t.url
}

// GetStartLevel returns the start level configuration from the TzktConfig.
func (t *TzktConfig) GetStartLevel() uint64 {
	return t.startLevel
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

// GetPort returns the port configuration from the MetricsConfig.
func (m *MetricsConfig) GetPort() int {
	return m.port
}

// GetPostgresqlDSN constructs a PostgreSQL DSN from the DBConfig.
func (d *DBConfig) GetPostgresqlDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", d.user, d.password, d.host, d.port, d.dbname)
}

// GetListenAddress constructs the listenning address from the ServerConfig.
func (s *ServerConfig) GetListenAddress() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}
