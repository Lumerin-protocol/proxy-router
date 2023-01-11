package lib

import (
	"os"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

/*
const blue = "\u001b[34m"
const green = "\u001b[32m"
const red = "\u001b[31m"
const reset = "\u001b[0m"
*/

type Logger struct {
	*zap.SugaredLogger
}

func (l *Logger) LogMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	l.Debugf("\tAlloc = %v MiB", bToMb(m.Alloc))
	l.Debugf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	l.Debugf("\tSys = %v MiB", bToMb(m.Sys))
	l.Debugf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func NewLogger(isProduction bool, level string, logToFile bool, color bool) (*Logger, error) {
	var (
		log *zap.Logger
		err error
	)

	if isProduction {
		log, err = newProductionLogger(level)
	} else {
		log, err = NewDevelopmentLogger(level, logToFile, color, true)
	}
	if err != nil {
		return nil, err
	}

	return &Logger{SugaredLogger: log.Sugar()}, nil
}

// NewTestLogger logs only to stdout
func NewTestLogger() *zap.SugaredLogger {
	log, _ := NewDevelopmentLogger("debug", false, false, false)
	return log.Sugar()
}

func NewDevelopmentLogger(levelStr string, logToFile bool, color bool, addCaller bool) (*zap.Logger, error) {
	consoleEncoderCfg := zap.NewDevelopmentEncoderConfig()
	consoleEncoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
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
		fileEncoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
		fileEncoder := zapcore.NewConsoleEncoder(fileEncoderCfg)

		file, err := os.OpenFile("logfile.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
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

func LogMsg(isMiner bool, isRead bool, addr string, payload []byte, l interface{}) {
	// return
	var (
		source string
		op     string
		// cut    int = 100
	)
	if isMiner {
		source = "MINER"
	} else {
		source = "POOL "
	}
	if isRead {
		op = "<-"
	} else {
		op = "->"
	}
	msg := string(payload)
	// if len(msg) > cut {
	// 	msg = msg[:cut] + "...}"
	// }
	// TODO: move this to logger implementation
	if zapLogger, ok := l.(*zap.SugaredLogger); ok {
		zapLogger.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar().Debugf("%s %s(%s): %s", source, op, addr, msg)
	}
}
