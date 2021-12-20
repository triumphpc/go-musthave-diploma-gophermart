package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

// New Config instance
func New() (*zap.Logger, error) {
	// Init config
	cfg := zap.NewProductionConfig()
	// Set level
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	// Log level
	atom := zap.NewAtomicLevel()
	// level
	level := zap.InfoLevel.String()
	if err := atom.UnmarshalText([]byte(level)); err != nil {
		return nil, err
	}
	cfg.Level = atom
	// Output set
	cfg.OutputPaths = []string{"stdout"}
	// Time format
	cfg.EncoderConfig.EncodeTime = customMillisTimeEncoder
	return cfg.Build()
}

// customMillisTimeEncoder set time format
func customMillisTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.UTC().Format("2006-01-02 15:04:05"))
}
