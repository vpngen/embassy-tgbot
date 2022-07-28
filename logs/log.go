package logs

import (
	"fmt"
	"os"
	"sync/atomic"
)

// Logging levels.
const (
	LevelCritical int32 = iota
	LevelError
	LevelWarning
	LevelInfo
	LevelDebug
)

// logLevel - global logging level.
var logLevel = LevelCritical //nolint

// SetLogLevel - set global logging level.
func SetLogLevel(n int32) {
	atomic.StoreInt32(&logLevel, n)
}

// LogLevel - get global logging level.
func LogLevel() int32 {
	return atomic.LoadInt32(&logLevel)
}

// Debug - print debug message.
func Debug(a ...any) {
	if LogLevel() == LevelDebug {
		fmt.Fprint(os.Stdout, a...)
	}
}

// Debugln - print debug message.
func Debugln(a ...any) {
	if LogLevel() == LevelDebug {
		fmt.Fprintln(os.Stdout, a...)
	}
}

// Debugf - print debug message.
func Debugf(format string, a ...any) {
	if LogLevel() == LevelDebug {
		fmt.Fprintf(os.Stdout, format, a...)
	}
}

// Info - print debug message.
func Info(a ...any) {
	if LogLevel() >= LevelInfo {
		fmt.Fprint(os.Stdout, a...)
	}
}

// Infoln - print debug message.
func Infoln(a ...any) {
	if LogLevel() >= LevelInfo {
		fmt.Fprintln(os.Stdout, a...)
	}
}

// Infof - print debug message.
func Infof(format string, a ...any) {
	if LogLevel() >= LevelInfo {
		fmt.Fprintf(os.Stdout, format, a...)
	}
}

// Warning - print debug message.
func Warning(a ...any) {
	if LogLevel() >= LevelWarning {
		fmt.Fprint(os.Stderr, a...)
	}
}

// Warningln - print debug message.
func Warningln(a ...any) {
	if LogLevel() >= LevelWarning {
		fmt.Fprintln(os.Stderr, a...)
	}
}

// Warningf - print debug message.
func Warningf(format string, a ...any) {
	if LogLevel() >= LevelWarning {
		fmt.Fprintf(os.Stderr, format, a...)
	}
}

// Err - print debug message.
func Err(a ...any) {
	if LogLevel() >= LevelError {
		fmt.Fprint(os.Stderr, a...)
	}
}

// Errln - print debug message.
func Errln(a ...any) {
	if LogLevel() >= LevelError {
		fmt.Fprintln(os.Stderr, a...)
	}
}

// Errf - print debug message.
func Errf(format string, a ...any) {
	if LogLevel() >= LevelError {
		fmt.Fprintf(os.Stderr, format, a...)
	}
}

// Critic - print debug message.
func Critic(a ...any) {
	fmt.Fprint(os.Stderr, a...)
}

// Criticln - print debug message.
func Criticln(a ...any) {
	fmt.Fprintln(os.Stderr, a...)
}

// Criticf - print debug message.
func Criticf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
}
