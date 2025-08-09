package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	DB     DBConfig     `mapstructure:"db"`
	OIDC   OIDCConfig   `mapstructure:"oidc"`
	Log    LogConfig    `mapstructure:"log"`
}

// ServerConfig holds server-specific configuration.
type ServerConfig struct {
	Port string     `mapstructure:"port"`
	TLS  TLSConfig  `mapstructure:"tls"`
}

// TLSConfig holds TLS-specific configuration.
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"certFile"`
	KeyFile  string `mapstructure:"keyFile"`
}

// DBConfig holds database-specific configuration.
type DBConfig struct {
	DSN string `mapstructure:"dsn"`
}

// OIDCConfig holds OIDC client configuration.
type OIDCConfig struct {
	IssuerURL    string `mapstructure:"issuer_url"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string `mapstructure:"level"`  // e.g., "debug", "info", "warn", "error"
	Format string `mapstructure:"format"` // e.g., "json", "console"
}

// LoadConfig reads configuration from file and environment variables.
func LoadConfig() (*Config, error) {
	// Set default values
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("db.dsn", "wiki.db")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "console")


	// Set up viper to read from config file
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/go-wiki-app/")
	viper.AddConfigPath("$HOME/.go-wiki-app")

	// Attempt to read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return nil, err
		}
		// Config file not found; proceed with defaults and env vars
	}

	// Set up viper to read from environment variables
	viper.SetEnvPrefix("WIKI")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Unmarshal the config into the Config struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
