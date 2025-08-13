package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	DB      DBConfig      `mapstructure:"db"`
	OIDC    OIDCConfig    `mapstructure:"oidc"`
	Log     LogConfig     `mapstructure:"log"`
	Session SessionConfig `mapstructure:"session"`
	Cache   CacheConfig   `mapstructure:"cache"`
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
	DSN                 string `mapstructure:"dsn"`
	MaxOpenConns        int    `mapstructure:"max_open_conns"`
	MaxIdleConns        int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeMins int    `mapstructure:"conn_max_lifetime_mins"`
	ConnMaxIdleTimeMins int    `mapstructure:"conn_max_idle_time_mins"`
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

// SessionConfig holds session management configuration.
type SessionConfig struct {
	SecretKey string `mapstructure:"secret_key"`
	Lifetime  int    `mapstructure:"lifetime_hours"`
}

// CacheConfig holds cache-specific configuration.
type CacheConfig struct {
	FilePath          string   `mapstructure:"file_path"`
	DefaultTTLSeconds int      `mapstructure:"default_ttl_seconds"`
	Pragmas           []string `mapstructure:"pragmas"`
}

// LoadConfig reads configuration from file and environment variables.
func LoadConfig() (*Config, error) {
	// Set default values
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("db.dsn", "wikiuser:wikipass@tcp(127.0.0.1:3306)/go_wiki_app?parseTime=true")
	viper.SetDefault("db.max_open_conns", 25)
	viper.SetDefault("db.max_idle_conns", 25)
	viper.SetDefault("db.conn_max_lifetime_mins", 5)
	viper.SetDefault("db.conn_max_idle_time_mins", 2)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "console")
	viper.SetDefault("session.lifetime_hours", 24)
	// No default for secret key, it must be provided.
	viper.SetDefault("cache.file_path", "cache.db")
	viper.SetDefault("cache.default_ttl_seconds", 300) // 5 minutes
	viper.SetDefault("cache.pragmas", []string{
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA temp_store = MEMORY;",
		"PRAGMA cache_size = -20000;",   // ~20MB
		"PRAGMA mmap_size = 268435456;", // 256MB
	})


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
