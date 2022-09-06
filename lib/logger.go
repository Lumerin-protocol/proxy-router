package lib

import (
	"os"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const blue = "\u001b[34m"
const green = "\u001b[32m"
const red = "\u001b[31m"
const reset = "\u001b[0m"

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

	opts := []zap.Option{
		zap.Development(),
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	}

	return zap.New(core, opts...), nil
}

func newProductionLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return l, nil
}

func LogMsg(isMiner bool, isRead bool, addr string, payload []byte, l interfaces.ILogger) {
	return
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
