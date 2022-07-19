package lib

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(syslog bool) (*zap.SugaredLogger, error) {
	var (
		log *zap.Logger
		err error
	)

	if syslog {
		log, err = newProductionLogger()
	} else {
		log, err = newDevelopmentLogger()
	}
	if err != nil {
		return nil, err
	}

	return log.Sugar(), nil
}

func newDevelopmentLogger() (*zap.Logger, error) {
	consoleEncoderCfg := zap.NewDevelopmentEncoderConfig()
	consoleEncoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	consoleEncoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderCfg)

	fileEncoderCfg := zap.NewDevelopmentEncoderConfig()
	fileEncoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	fileEncoder := zapcore.NewConsoleEncoder(fileEncoderCfg)

	file, err := os.OpenFile("logfile.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, zapcore.AddSync(file), zap.DebugLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel),
	)

	return zap.New(core), nil
}

func newProductionLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return l, nil
}
