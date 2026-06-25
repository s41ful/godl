package logger

import (
	"log"
	"strings"
)

type LOG_LEVEL int

const (
	LOG_LEVEL_NONE = iota
	LOG_LEVEL_INFO
	LOG_LEVEL_WARN
	LOG_LEVEL_DEBUG
)

const (
	LOG_LEVEL_NONE_STR  = "none"
	LOG_LEVEL_INFO_STR  = "info"
	LOG_LEVEL_WARN_STR  = "warn"
	LOG_LEVEL_DEBUG_STR = "debug"
)

type Logger struct {
	logLevel LOG_LEVEL
}

func NewLogger(logLevel string) *Logger {
	logLevel = strings.ToLower(logLevel)

	switch logLevel {
	case LOG_LEVEL_NONE_STR:
		return &Logger{
			logLevel: LOG_LEVEL_NONE,
		}
	case LOG_LEVEL_INFO_STR:
		return &Logger{
			logLevel: LOG_LEVEL_INFO,
		}
	case LOG_LEVEL_WARN_STR:
		return &Logger{
			logLevel: LOG_LEVEL_WARN,
		}
	case LOG_LEVEL_DEBUG_STR:
		return &Logger{
			logLevel:  LOG_LEVEL_DEBUG,
		}
	default:
		return &Logger{
			logLevel:  LOG_LEVEL_INFO,
		}

	}
}

func (l *Logger) SetLogLevel(level LOG_LEVEL) {
	l.logLevel = level
}

func (l *Logger) SetFlags(flag int) {
	log.SetFlags(flag)
}

func (l *Logger) Print(level LOG_LEVEL, v any) {
	if level <= l.logLevel {
		log.Print(v)
	}
}

func (l *Logger) Printf(level LOG_LEVEL, format string, v ...any) {
	if level <= l.logLevel {
		log.Printf(format, v...)
	}
}

func (l *Logger) Println(level LOG_LEVEL, v any) {
	if level <= l.logLevel {
		log.Println(v)
	}
}

func (l *Logger) Fatal(v any) {
	log.Fatal(v)
}

func (l *Logger) Fatalf(format string, v ...any) {
	log.Fatalf(format, v...)
}

func (l *Logger) Fatalln(v any) {
	log.Fatalln(v)
}
