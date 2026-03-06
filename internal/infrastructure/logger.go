package infrastructure

import (
	"go-standard/internal/apperror"
	"go-standard/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger builds a zap.Logger from config, replaces the global logger, and
// returns a cleanup function that flushes buffered log entries.
// Satisfies the Wire (T, func(), error) cleanup pattern.
func NewLogger(cfg *config.Config) (*zap.Logger, func(), error) {
	level, err := parseLevel(cfg.Log.Level)
	if err != nil {
		return nil, nil, apperror.Internal("logger: invalid log level", err)
	}

	zapCfg := buildZapConfig(cfg.App.Env, cfg.Log.Format, level)

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, nil, apperror.Internal("logger: build failed", err)
	}

	zap.ReplaceGlobals(logger)

	logger.Info("logger: initialized",
		zap.String("level", cfg.Log.Level),
		zap.String("format", cfg.Log.Format),
		zap.String("env", cfg.App.Env),
	)

	cleanup := func() {
		// Sync flushes any buffered log entries. The error is intentionally
		// ignored here — common on stderr/stdout targets (os.Exit race).
		_ = logger.Sync()
	}

	return logger, cleanup, nil
}

// parseLevel converts a string log level to zap.AtomicLevel.
func parseLevel(level string) (zap.AtomicLevel, error) {
	atomicLevel := zap.NewAtomicLevel()
	if err := atomicLevel.UnmarshalText([]byte(level)); err != nil {
		return atomicLevel, err
	}
	return atomicLevel, nil
}

// buildZapConfig returns a production config for prod/staging and a
// development config for all other environments.
func buildZapConfig(env, format string, level zap.AtomicLevel) zap.Config {
	var zapCfg zap.Config

	switch env {
	case "prod", "staging":
		zapCfg = zap.NewProductionConfig()
	default:
		zapCfg = zap.NewDevelopmentConfig()
	}

	zapCfg.Level = level
	zapCfg.DisableStacktrace = true

	encoding := format
	if encoding == "" {
		encoding = "json"
	}
	zapCfg.Encoding = encoding

	// Ensure console encoding uses the same human-readable key names as JSON.
	if encoding == "json" {
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	return zapCfg
}
