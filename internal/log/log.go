package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

const appName = "noct"

type Logger struct {
	mu   sync.Mutex
	file *os.File
	lvl  Level
}

var (
	std     *Logger
	stdOnce sync.Once
)

func Setup(level Level) error {
	var err error
	stdOnce.Do(func() {
		dir, dirErr := configDir()
		if dirErr != nil {
			err = fmt.Errorf("log directory: %w", dirErr)
			return
		}
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			err = fmt.Errorf("create log directory: %w", mkErr)
			return
		}

		path := filepath.Join(dir, appName+".log")
		f, openErr := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if openErr != nil {
			err = fmt.Errorf("open log file %s: %w", path, openErr)
			return
		}

		std = &Logger{file: f, lvl: level}
	})
	return err
}

func Default() *Logger {
	return std
}

func (l *Logger) Debugf(format string, args ...any) {
	l.logf(Debug, format, args...)
}

func (l *Logger) Infof(format string, args ...any) {
	l.logf(Info, format, args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.logf(Warn, format, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.logf(Error, format, args...)
}

func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

func (l *Logger) logf(lvl Level, format string, args ...any) {
	if l == nil {
		return
	}
	if lvl < l.lvl {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return
	}

	prefix := map[Level]string{
		Debug: "DBG",
		Info:  "INF",
		Warn:  "WRN",
		Error: "ERR",
	}[lvl]

	fmt.Fprintf(l.file, "[%s] %s\n", prefix, fmt.Sprintf(format, args...))
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}
