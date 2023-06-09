package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const timeLayout = "2006-01-02T15:04:05"

type Logger struct {
	*zap.SugaredLogger
}

func (l *Logger) Named(name string) interfaces.ILogger {
	return &Logger{l.SugaredLogger.Named(name)}
}

func (l *Logger) With(args ...interface{}) interfaces.ILogger {
	return &Logger{l.SugaredLogger.With(args...)}
}

func NewLogger(isProduction bool, level string, logToFile bool, color bool) (*Logger, error) {
	var (
		log *zap.Logger
		err error
	)

	if isProduction {
		log, err = newProductionLogger(level)
	} else {
		log, err = NewDevelopmentLogger(level, logToFile, color, false)
	}
	if err != nil {
		return nil, err
	}

	return &Logger{SugaredLogger: log.Sugar()}, nil
}

// NewTestLogger logs only to stdout
func NewTestLogger() *Logger {
	log, _ := NewDevelopmentLogger("debug", false, false, false)
	return &Logger{SugaredLogger: log.Sugar()}
}

func NewDevelopmentLogger(levelStr string, logToFile bool, color bool, addCaller bool) (*zap.Logger, error) {
	consoleEncoderCfg := zap.NewDevelopmentEncoderConfig()
	consoleEncoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout(timeLayout)
	if color {
		consoleEncoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderCfg)

	var core zapcore.Core
	level, err := zapcore.ParseLevel(levelStr)
	if err != nil {
		return nil, err
	}

	if logToFile {
		fileEncoderCfg := zap.NewDevelopmentEncoderConfig()
		fileEncoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout(timeLayout)
		fileEncoder := zapcore.NewConsoleEncoder(fileEncoderCfg)

		newpath := filepath.Join(".", "logs")
		err := os.MkdirAll(newpath, os.ModePerm)
		if err != nil {
			return nil, err
		}
		file, err := os.OpenFile("./logs/logfile.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return nil, err
		}

		core = zapcore.NewTee(
			zapcore.NewCore(fileEncoder, zapcore.AddSync(file), level),
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
		)
	} else {
		core = zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level)
	}

	opts := []zap.Option{
		zap.Development(),
		zap.AddStacktrace(zap.ErrorLevel),
	}
	if addCaller {
		opts = append(opts, zap.AddCaller())
	}

	return zap.New(core, opts...), nil
}

func newProductionLogger(levelStr string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	level, err := zapcore.ParseLevel(levelStr)
	if err != nil {
		return nil, err
	}
	cfg.Level = zap.NewAtomicLevelAt(level)
	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return l, nil
}

func NewFileLogger(name string) (*zap.SugaredLogger, error) {
	fileEncoderCfg := zap.NewDevelopmentEncoderConfig()
	fileEncoderCfg.LevelKey = zapcore.OmitKey
	fileEncoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	fileEncoder := zapcore.NewConsoleEncoder(fileEncoderCfg)

	path := filepath.Join(".", "logs", "protocol")
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	filename := SanitizeFilename(fmt.Sprintf("%s-%s", time.Now().Format(timeLayout), name))
	pathName := fmt.Sprintf("%s/%s.log", path, filename)

	file, err := os.OpenFile(pathName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	core := zapcore.NewCore(fileEncoder, zapcore.AddSync(file), zap.DebugLevel)
	return zap.New(core).Sugar(), nil
}
