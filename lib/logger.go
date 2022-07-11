package lib

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(syslog bool) (*zap.SugaredLogger, error) {
	var cfg zap.Config

	if syslog {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = newDevelopmentConfig()
	}

	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return l.Sugar(), nil
}

func newDevelopmentConfig() zap.Config {
	cfg := zap.NewDevelopmentConfig()
	cfg.Development = false
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.OutputPaths = []string{"logfile.log", "stdout"}
	return cfg
}
