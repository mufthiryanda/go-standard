package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config is the root configuration struct loaded at boot.
type Config struct {
	App           App                `mapstructure:"app"`
	DB            DB                 `mapstructure:"db"`
	Redis         Redis              `mapstructure:"redis"`
	Elasticsearch Elasticsearch      `mapstructure:"elasticsearch"`
	JWT           JWT                `mapstructure:"jwt"`
	RateLimit     RateLimit          `mapstructure:"rate_limit"`
	Log           Log                `mapstructure:"log"`
	Integrations  IntegrationsConfig `mapstructure:"integrations"`
}

// App holds general application settings.
type App struct {
	Name     string `mapstructure:"name"`
	Port     int    `mapstructure:"port"`
	Env      string `mapstructure:"env"`
	Timezone string `mapstructure:"timezone"`
}

// DB holds PostgreSQL connection settings.
type DB struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Name            string `mapstructure:"name"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	SSLMode         string `mapstructure:"ssl_mode"`
	MaxOpenConn     int    `mapstructure:"max_open_conns"`
	MaxIdleConn     int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime string `mapstructure:"conn_max_lifetime"`
}

// Redis holds Redis connection settings.
type Redis struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Password    string `mapstructure:"password"`
	DB          int    `mapstructure:"db"`
	PoolSize    int    `mapstructure:"pool_size"`
	MinIdleConn int    `mapstructure:"min_idle_conns"`
}

// Elasticsearch holds ES client settings.
type Elasticsearch struct {
	Addresses []string `mapstructure:"addresses"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
}

// JWT holds token signing settings.
type JWT struct {
	Secret     string `mapstructure:"secret"`
	AccessTTL  string `mapstructure:"access_ttl"`
	RefreshTTL string `mapstructure:"refresh_ttl"`
	Issuer     string `mapstructure:"issuer"`
}

// RateLimitPolicy holds max requests and window for a single policy.
type RateLimitPolicy struct {
	Max    int    `mapstructure:"max"`
	Window string `mapstructure:"window"`
}

// RateLimit groups rate limit policies.
type RateLimit struct {
	Default RateLimitPolicy `mapstructure:"default"`
	Auth    RateLimitPolicy `mapstructure:"auth"`
}

// Log holds logging settings.
type Log struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// IntegrationsConfig holds all third-party integration configurations.
type IntegrationsConfig struct {
	SnapBI SnapBI `mapstructure:"snap_bi"`
}

// SnapBI holds configuration for the SNAP BI (Bank Nasional Indonesia) integration.
type SnapBI struct {
	BaseURL        string `mapstructure:"base_url"`
	ClientKey      string `mapstructure:"client_key"`
	PartnerID      string `mapstructure:"partner_id"`
	ChannelID      string `mapstructure:"channel_id"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
	// ClientSecret is env-only — never store in YAML.
	// Env var: APP_INTEGRATIONS_SNAP_BI_CLIENT_SECRET
	ClientSecret   string `mapstructure:"-"`
	AccessTokenTTL int    `mapstructure:"access_token_ttl"` // seconds, default 840
}

// replacer maps APP_DB_HOST → db.host for Viper env binding.
func replacer() *strings.Replacer {
	return strings.NewReplacer(".", "_")
}

// LoadConfig reads config.{APP_ENV}.yaml and merges APP_* env vars.
func LoadConfig() (*Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	v := viper.New()
	v.SetConfigName(fmt.Sprintf("config.%s", env))
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(replacer())
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config: read failed: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal failed: %w", err)
	}

	return &cfg, nil
}
