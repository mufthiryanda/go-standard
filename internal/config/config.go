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
	Storage       StorageConfig      `mapstructure:"storage"`
	Worker        WorkerConfig       `mapstructure:"worker"`
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

// StorageConfig holds all object storage configuration.
type StorageConfig struct {
	ActiveProvider    string           `mapstructure:"active_provider"` // "s3" | "minio" | "do_spaces"
	DefaultBucket     string           `mapstructure:"default_bucket"`
	PublicBaseURL     string           `mapstructure:"public_base_url"`     // CDN base, e.g. https://cdn.example.com
	PresignedGetTTL   int              `mapstructure:"presigned_get_ttl"`   // seconds, default: 3600 (1h)
	PresignedPutTTL   int              `mapstructure:"presigned_put_ttl"`   // seconds, default: 900 (15min)
	MaxFileSizeBytes  int64            `mapstructure:"max_file_size_bytes"` // default: 10485760 (10MB)
	AllowedMIMETypes  []string         `mapstructure:"allowed_mime_types"`
	AllowedExtensions []string         `mapstructure:"allowed_extensions"` // include dot: [".jpg", ".pdf"]
	Providers         StorageProviders `mapstructure:"providers"`
}

// StorageProviders holds per-provider configuration.
type StorageProviders struct {
	S3       S3Config       `mapstructure:"s3"`
	MinIO    MinIOConfig    `mapstructure:"minio"`
	DOSpaces DOSpacesConfig `mapstructure:"do_spaces"`
}

// S3Config configures AWS S3.
type S3Config struct {
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"-"` // env: APP_STORAGE_PROVIDERS_S3_ACCESS_KEY_ID
	SecretAccessKey string `mapstructure:"-"` // env: APP_STORAGE_PROVIDERS_S3_SECRET_ACCESS_KEY
}

// MinIOConfig configures a self-hosted MinIO instance.
type MinIOConfig struct {
	Endpoint        string `mapstructure:"endpoint"` // e.g. "http://localhost:9000"
	UseSSL          bool   `mapstructure:"use_ssl"`
	AccessKeyID     string `mapstructure:"-"` // env: APP_STORAGE_PROVIDERS_MINIO_ACCESS_KEY_ID
	SecretAccessKey string `mapstructure:"-"` // env: APP_STORAGE_PROVIDERS_MINIO_SECRET_ACCESS_KEY
}

// DOSpacesConfig configures DigitalOcean Spaces.
// Endpoint is auto-derived: https://{region}.digitaloceanspaces.com
type DOSpacesConfig struct {
	Region          string `mapstructure:"region"` // e.g. "sgp1", "nyc3"
	AccessKeyID     string `mapstructure:"-"`      // env: APP_STORAGE_PROVIDERS_DO_SPACES_ACCESS_KEY_ID
	SecretAccessKey string `mapstructure:"-"`      // env: APP_STORAGE_PROVIDERS_DO_SPACES_SECRET_ACCESS_KEY
}

// WorkerConfig holds configuration for the asynq worker binary.
type WorkerConfig struct {
	Concurrency   int          `mapstructure:"concurrency"` // default: 10
	Queues        WorkerQueues `mapstructure:"queues"`
	RetentionDays int          `mapstructure:"retention_days"` // archived task retention, default: 7
}

// WorkerQueues maps logical domain names to their asynq queue names.
type WorkerQueues struct {
	User         string `mapstructure:"user"`         // default: "user"
	Payment      string `mapstructure:"payment"`      // default: "payment"
	Notification string `mapstructure:"notification"` // default: "notification"
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
