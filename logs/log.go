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
var logLevel int32 = LevelCritical

// SetLogLevel - set global logging level.
func SetLogLevel(n int32) {
	atomic.StoreInt32(&logLevel, n)
}

// LogLevel - get global logging level.
func LogLevel() int32 {
	return atomic.LoadInt32(&logLevel)
}

// Debug - print debug message.
func Debug(a ...any) (int, error) {
	if LogLevel() == LevelDebug {
		return fmt.Fprint(os.Stdout, a...)
	}

	return 0, nil
}

// Debugln - print debug message.
func Debugln(a ...any) (int, error) {
	if LogLevel() == LevelDebug {
		return fmt.Fprintln(os.Stdout, a...)
	}

	return 0, nil
}

// Debugf - print debug message.
func Debugf(format string, a ...any) (int, error) {
	if LogLevel() == LevelDebug {
		return fmt.Fprintf(os.Stdout, format, a...)
	}

	return 0, nil
}

// Info - print debug message.
func Info(a ...any) (int, error) {
	if LogLevel() >= LevelInfo {
		return fmt.Fprint(os.Stdout, a...)
	}

	return 0, nil
}

// Infoln - print debug message.
func Infoln(a ...any) (int, error) {
	if LogLevel() >= LevelInfo {
		return fmt.Fprintln(os.Stdout, a...)
	}

	return 0, nil
}

// Infof - print debug message.
func Infof(format string, a ...any) (int, error) {
	if LogLevel() >= LevelInfo {
		return fmt.Fprintf(os.Stdout, format, a...)
	}

	return 0, nil
}

// Warning - print debug message.
func Warning(a ...any) (int, error) {
	if LogLevel() >= LevelWarning {
		return fmt.Fprint(os.Stderr, a...)
	}

	return 0, nil
}

// Warningln - print debug message.
func Warningln(a ...any) (int, error) {
	if LogLevel() >= LevelWarning {
		return fmt.Fprintln(os.Stderr, a...)
	}

	return 0, nil
}

// Warningf - print debug message.
func Warningf(format string, a ...any) (int, error) {
	if LogLevel() >= LevelWarning {
		return fmt.Fprintf(os.Stderr, format, a...)
	}

	return 0, nil
}

// Err - print debug message.
func Err(a ...any) (int, error) {
	if LogLevel() >= LevelError {
		return fmt.Fprint(os.Stderr, a...)
	}

	return 0, nil
}

// Errln - print debug message.
func Errln(a ...any) (int, error) {
	if LogLevel() >= LevelError {
		return fmt.Fprintln(os.Stderr, a...)
	}

	return 0, nil
}

// Errf - print debug message.
func Errf(format string, a ...any) (int, error) {
	if LogLevel() >= LevelError {
		return fmt.Fprintf(os.Stderr, format, a...)
	}

	return 0, nil
}

// Critic - print debug message.
func Critic(a ...any) (int, error) {
	return fmt.Fprint(os.Stderr, a...)
}

// Criticln - print debug message.
func Criticln(a ...any) (int, error) {
	return fmt.Fprintln(os.Stderr, a...)
}

// Criticf - print debug message.
func Criticf(format string, a ...any) (int, error) {
	return fmt.Fprintf(os.Stderr, format, a...)
}
