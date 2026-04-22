package omlog

import (
	"encoding/json"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	mu      sync.Mutex
	file    *os.File
	path    string
	verbose *stdlog.Logger
}

func New(verbose io.Writer) (*Logger, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	logDir := filepath.Join(home, "Library", "Logs", "OpenMigrate")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}
	name := time.Now().Format("20060102-150405") + ".log"
	path := filepath.Join(logDir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	var verboseLogger *stdlog.Logger
	if verbose != nil {
		verboseLogger = stdlog.New(verbose, "", 0)
	}
	return &Logger{file: file, path: path, verbose: verboseLogger}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Logger) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.write("info", msg, fields)
}

func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.write("warn", msg, fields)
}

func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.write("error", msg, fields)
}

func (l *Logger) write(level, msg string, fields map[string]interface{}) {
	if l == nil {
		return
	}
	record := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level,
		"message":   msg,
	}
	for k, v := range fields {
		if isSensitiveKey(k) {
			continue
		}
		switch s := v.(type) {
		case string:
			if looksSensitive(s) {
				continue
			}
		}
		record[k] = v
	}
	line, err := json.Marshal(record)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.file.Write(append(line, '\n'))
	if l.verbose != nil {
		l.verbose.Printf("%s %s", strings.ToUpper(level), msg)
	}
}

func isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "passphrase") || strings.Contains(key, "secret") || strings.Contains(key, "token")
}

func looksSensitive(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "passphrase") || strings.Contains(value, "-----begin age encrypted file-----")
}

func RedactPassphrase(fields map[string]interface{}) map[string]interface{} {
	clean := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		if isSensitiveKey(k) {
			continue
		}
		clean[k] = v
	}
	return clean
}

func MustLogger(verbose io.Writer) *Logger {
	logger, err := New(verbose)
	if err != nil {
		panic(fmt.Sprintf("create logger: %v", err))
	}
	return logger
}
