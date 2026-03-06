package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

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

type App struct {
	Name     string `mapstructure:"name"`
	Port     int    `mapstructure:"port"`
	Env      string `mapstructure:"env"`
	Timezone string `mapstructure:"timezone"`
}

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

type Redis struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Password    string `mapstructure:"password"`
	DB          int    `mapstructure:"db"`
	PoolSize    int    `mapstructure:"pool_size"`
	MinIdleConn int    `mapstructure:"min_idle_conns"`
}

type Elasticsearch struct {
	Addresses []string `mapstructure:"addresses"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
}

type JWT struct {
	Secret     string `mapstructure:"secret"`
	AccessTTL  string `mapstructure:"access_ttl"`
	RefreshTTL string `mapstructure:"refresh_ttl"`
	Issuer     string `mapstructure:"issuer"`
}

type RateLimitPolicy struct {
	Max    int    `mapstructure:"max"`
	Window string `mapstructure:"window"`
}

type RateLimit struct {
	Default RateLimitPolicy `mapstructure:"default"`
	Auth    RateLimitPolicy `mapstructure:"auth"`
}

type Log struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type IntegrationsConfig struct {
	SnapBI SnapBI `mapstructure:"snap_bi"`
}

type SnapBI struct {
	BaseURL        string `mapstructure:"base_url"`
	ClientKey      string `mapstructure:"client_key"`
	PartnerID      string `mapstructure:"partner_id"`
	ChannelID      string `mapstructure:"channel_id"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
	ClientSecret   string `mapstructure:"-"`
	AccessTokenTTL int    `mapstructure:"access_token_ttl"`
}

type StorageConfig struct {
	ActiveProvider    string           `mapstructure:"active_provider"`
	DefaultBucket     string           `mapstructure:"default_bucket"`
	PublicBaseURL     string           `mapstructure:"public_base_url"`
	PresignedGetTTL   int              `mapstructure:"presigned_get_ttl"`
	PresignedPutTTL   int              `mapstructure:"presigned_put_ttl"`
	MaxFileSizeBytes  int64            `mapstructure:"max_file_size_bytes"`
	AllowedMIMETypes  []string         `mapstructure:"allowed_mime_types"`
	AllowedExtensions []string         `mapstructure:"allowed_extensions"`
	Providers         StorageProviders `mapstructure:"providers"`
}

type StorageProviders struct {
	S3       S3Config       `mapstructure:"s3"`
	MinIO    MinIOConfig    `mapstructure:"minio"`
	DOSpaces DOSpacesConfig `mapstructure:"do_spaces"`
}

type S3Config struct {
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"-"`
	SecretAccessKey string `mapstructure:"-"`
}

type MinIOConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	UseSSL          bool   `mapstructure:"use_ssl"`
	AccessKeyID     string `mapstructure:"-"`
	SecretAccessKey string `mapstructure:"-"`
}

type DOSpacesConfig struct {
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"-"`
	SecretAccessKey string `mapstructure:"-"`
}

type WorkerConfig struct {
	Concurrency   int          `mapstructure:"concurrency"`
	Queues        WorkerQueues `mapstructure:"queues"`
	RetentionDays int          `mapstructure:"retention_days"`
}

type WorkerQueues struct {
	User         string `mapstructure:"user"`
	Payment      string `mapstructure:"payment"`
	Notification string `mapstructure:"notification"`
}

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
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicit binding ensures Unmarshal picks up env vars for secret keys.
	_ = v.BindEnv("db.password", "APP_DB_PASSWORD")
	_ = v.BindEnv("redis.password", "APP_REDIS_PASSWORD")
	_ = v.BindEnv("jwt.secret", "APP_JWT_SECRET")
	_ = v.BindEnv("elasticsearch.username", "APP_ES_USERNAME")
	_ = v.BindEnv("elasticsearch.password", "APP_ES_PASSWORD")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config: read failed: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal failed: %w", err)
	}

	// mapstructure:"-" fields must be loaded manually via os.Getenv.
	cfg.Integrations.SnapBI.ClientSecret = os.Getenv("APP_INTEGRATIONS_SNAP_BI_CLIENT_SECRET")
	cfg.Storage.Providers.MinIO.AccessKeyID = os.Getenv("APP_STORAGE_PROVIDERS_MINIO_ACCESS_KEY_ID")
	cfg.Storage.Providers.MinIO.SecretAccessKey = os.Getenv("APP_STORAGE_PROVIDERS_MINIO_SECRET_ACCESS_KEY")
	cfg.Storage.Providers.S3.AccessKeyID = os.Getenv("APP_STORAGE_PROVIDERS_S3_ACCESS_KEY_ID")
	cfg.Storage.Providers.S3.SecretAccessKey = os.Getenv("APP_STORAGE_PROVIDERS_S3_SECRET_ACCESS_KEY")
	cfg.Storage.Providers.DOSpaces.AccessKeyID = os.Getenv("APP_STORAGE_PROVIDERS_DO_SPACES_ACCESS_KEY_ID")
	cfg.Storage.Providers.DOSpaces.SecretAccessKey = os.Getenv("APP_STORAGE_PROVIDERS_DO_SPACES_SECRET_ACCESS_KEY")

	return &cfg, nil
}
