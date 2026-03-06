package infrastructure

import (
	"fmt"
	"log"
	"time"

	"go-standard/internal/apperror"
	"go-standard/internal/config"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// NewPostgresDB opens a GORM connection backed by pgx, configures the
// connection pool from cfg, and returns a cleanup function that closes the
// underlying *sql.DB. Satisfies the Wire (T, func(), error) cleanup pattern.
func NewPostgresDB(cfg *config.Config) (*gorm.DB, func(), error) {
	dsn := buildDSN(cfg.DB)

	log.Println("DB Password -> ", cfg.DB.Password)

	gormCfg := &gorm.Config{
		// Silence GORM's own logger; structured request logging happens in middleware.
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormCfg)
	if err != nil {
		return nil, nil, apperror.ServiceUnavailable("postgres: failed to open connection", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, apperror.ServiceUnavailable("postgres: failed to retrieve sql.DB", err)
	}

	if err = applyPoolSettings(sqlDB, cfg.DB); err != nil {
		return nil, nil, apperror.ServiceUnavailable("postgres: failed to configure pool", err)
	}

	if err = sqlDB.Ping(); err != nil {
		return nil, nil, apperror.ServiceUnavailable("postgres: ping failed", err)
	}

	zap.L().Info("postgres: connection established",
		zap.String("host", cfg.DB.Host),
		zap.Int("port", cfg.DB.Port),
		zap.String("database", cfg.DB.Name),
	)

	cleanup := func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			zap.L().Error("postgres: error closing connection", zap.Error(closeErr))
			return
		}
		zap.L().Info("postgres: connection closed")
	}

	return db, cleanup, nil
}

// buildDSN constructs the PostgreSQL DSN from config fields.
func buildDSN(db config.DB) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		db.Host, db.Port, db.User, db.Password, db.Name, db.SSLMode,
	)
}

// applyPoolSettings configures connection pool limits from config.
// Falls back to standard defaults when config values are zero.
func applyPoolSettings(sqlDB interface {
	SetMaxOpenConns(int)
	SetMaxIdleConns(int)
	SetConnMaxLifetime(time.Duration)
}, db config.DB) error {
	maxOpen := db.MaxOpenConn
	if maxOpen <= 0 {
		maxOpen = 25
	}

	maxIdle := db.MaxIdleConn
	if maxIdle <= 0 {
		maxIdle = 10
	}

	lifetime := 5 * time.Minute
	if db.ConnMaxLifetime != "" {
		parsed, err := time.ParseDuration(db.ConnMaxLifetime)
		if err != nil {
			return fmt.Errorf("postgres: invalid conn_max_lifetime %q: %w", db.ConnMaxLifetime, err)
		}
		lifetime = parsed
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(lifetime)

	return nil
}
