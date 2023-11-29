package lib

import (
	"io"
	"os"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const timeLayout = "2006-01-02T15:04:05"

func NewLogger(level string, color, isProd bool, isJSON bool, filepath string) (*Logger, error) {
	log, err := newLogger(level, color, isProd, isJSON, filepath, nil)

	if err != nil {
		return nil, err
	}

	return &Logger{SugaredLogger: log.Sugar()}, nil
}

func NewLoggerMemory(level string, color, isProd bool, isJSON bool, filepath string, wr io.Writer) (*Logger, error) {
	log, err := newLogger(level, color, isProd, isJSON, filepath, wr)

	if err != nil {
		return nil, err
	}

	return &Logger{SugaredLogger: log.Sugar()}, nil
}

// NewTestLogger logs only to stdout
func NewTestLogger() *Logger {
	log, _ := newLogger("debug", false, false, false, "", nil)
	return &Logger{SugaredLogger: log.Sugar()}
}

func newLogger(levelStr string, color bool, isProd bool, isJSON bool, filepath string, extraWriter io.Writer) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(levelStr)
	if err != nil {
		return nil, err
	}

	var cores []zapcore.Core

	if filepath != "" {
		fileCore, err := newFileCore(zapcore.DebugLevel, isProd, isJSON, filepath)
		if err != nil {
			return nil, err
		}
		cores = append(cores, fileCore)
	}
	if extraWriter != nil {
		memoryCore := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.AddSync(extraWriter), level)
		cores = append(cores, memoryCore)
	}

	consoleCore := newConsoleCore(level, color, isProd, isJSON)
	cores = append(cores, consoleCore)

	var core zapcore.Core
	if len(cores) > 1 {
		core = zapcore.NewTee(cores...)
	} else {
		core = cores[0]
	}

	opts := []zap.Option{
		zap.AddStacktrace(zap.ErrorLevel),
	}
	if !isProd {
		opts = append(opts, zap.Development())
	}

	return zap.New(core, opts...), nil
}

func newConsoleCore(level zapcore.Level, color bool, isProd bool, isJSON bool) zapcore.Core {
	encoderCfg := newEncoderCfg(isProd, color, isJSON)

	var encoder zapcore.Encoder
	if isJSON {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}
	return zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
}

func newEncoderCfg(isProd bool, color bool, isJSON bool) zapcore.EncoderConfig {
	var encoderCfg zapcore.EncoderConfig
	if isProd {
		encoderCfg = zap.NewProductionEncoderConfig()
	} else {
		encoderCfg = zap.NewDevelopmentEncoderConfig()
		encoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout(timeLayout)
	}

	if color && !isJSON {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	return encoderCfg
}

func newFileCore(level zapcore.Level, isProd bool, isJSON bool, path string) (zapcore.Core, error) {
	encoderCfg := newEncoderCfg(isProd, false, isJSON)
	if !isJSON {
		encoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout(timeLayout)
	}

	var encoder zapcore.Encoder
	if isJSON {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return zapcore.NewCore(encoder, zapcore.AddSync(file), level), nil
}

type Logger struct {
	*zap.SugaredLogger
}

func (l *Logger) Named(name string) interfaces.ILogger {
	return &Logger{l.SugaredLogger.Named(name)}
}

func (l *Logger) With(args ...interface{}) interfaces.ILogger {
	return &Logger{l.SugaredLogger.With(args...)}
}
