// log package helpers to enforce consistent log entries
// uses logrus
// greatly inspired by the infamous Scott Millner
package log

import (
	"io"
	"os"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

type Fields map[string]interface{}
type Level uint32
type Format string

const (
	// LevelPanic level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	LevelPanic Level = iota
	// LevelFatal level. Logs and then calls `logger.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	LevelFatal
	// LevelError level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	LevelError
	// LevelWarn level. Non-critical entries that deserve eyes.
	LevelWarn
	// LevelInfo level. General operational entries about what's going on inside the
	// application.
	LevelInfo
	// LevelDebug level. Usually only enabled when debugging. Very verbose logging.
	LevelDebug
	// LevelTrace level. Designates finer-grained informational events than the Debug.
	LevelTrace

	// FormatJSON will log entries in JSON format.
	FormatJSON = Format("json")
	// FormatInline will log entires inline.
	FormatInline = Format("inline")
)

type Logger struct {
	logger *logrus.Logger
}

var defaultOutput = os.Stdout

// New returns a new logger.
// Default output is os.Stdout.
func New() *Logger {
	l := logrus.New()
	l.SetOutput(defaultOutput)
	l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	return &Logger{
		l,
	}
}

// SetLevel sets the logging level for the logger.
func (l *Logger) SetLevel(level Level) *Logger {
	l.logger.SetLevel(logrus.Level(level))

	return l
}

// SetOutput sets the output for the logger.
func (l *Logger) SetOutput(output io.Writer) *Logger {
	l.logger.SetOutput(output)

	return l
}

// SetFormatJSON sets the logger to JSON format.
func (l *Logger) SetFormat(format Format) *Logger {
	l.logger.SetFormatter(&logrus.JSONFormatter{})

	return l
}

// Logf logs inline output at a given level.
func (l *Logger) Logf(level Level, format string, args ...interface{}) {
	switch level {
	case LevelInfo:
		l.logger.Infof(format, args...)
	case LevelTrace:
		l.logger.Tracef(format, args...)
	case LevelWarn:
		l.logger.Warnf(format, args...)
	case LevelDebug:
		l.logger.Debugf(format, args...)
	case LevelError:
		l.logger.Errorf(format, args...)
	case LevelPanic:
		l.logger.Panicf(format, args...)
	case LevelFatal:
		l.logger.Fatalf(format, args...)
	default:
		panic("invalid log level provided")
	}
}

// LogWithFields logs an event with fields.
func (l *Logger) LogWithFields(level Level, fields Fields, args ...interface{}) {
	switch level {
	case LevelInfo:
		l.withFields(fields).Info(args...)
	case LevelTrace:
		l.withFields(fields).Trace(args...)
	case LevelWarn:
		l.withFields(fields).Warn(args...)
	case LevelDebug:
		l.withFields(fields).Debug(args...)
	case LevelError:
		l.withFields(fields).Error(args...)
	case LevelPanic:
		l.withFields(fields).Panic(args...)
	case LevelFatal:
		l.withFields(fields).Fatal(args...)
	default:
		panic("invalid log level provided")
	}
}

func (l *Logger) withFields(fields Fields) *logrus.Entry {
	fields["stacktrace"] = string(debug.Stack())

	return l.logger.WithFields(logrus.Fields(fields))
}
