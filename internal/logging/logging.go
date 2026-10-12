package logging

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const logDateLayout = "2006-01-02"

// Loggers separates human-readable request output from structured error records.
type Loggers struct {
	Requests zerolog.Logger
	Errors   zerolog.Logger
	errorLog *dailyFile
}

// New creates a CLI request logger and an error logger with three-day retention.
func New(errorLogPath string) (*Loggers, error) {
	errorLog, err := newDailyFile(errorLogPath)
	if err != nil {
		return nil, err
	}

	console := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	requests := zerolog.New(console).With().Timestamp().Logger().Level(zerolog.InfoLevel)
	errors := zerolog.New(errorLog).With().Timestamp().Logger().Level(zerolog.ErrorLevel)

	return &Loggers{Requests: requests, Errors: errors, errorLog: errorLog}, nil
}

// Close stops the midnight rotation task and closes the active error log.
func (l *Loggers) Close() error {
	return l.errorLog.Close()
}

// dailyFile rotates error logs at local midnight and keeps three calendar days.
type dailyFile struct {
	basePath string
	day      string
	file     *os.File
	mu       sync.Mutex
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

func newDailyFile(basePath string) (*dailyFile, error) {
	if err := os.MkdirAll(filepath.Dir(basePath), 0o750); err != nil {
		return nil, err
	}
	if err := migrateLegacyFile(basePath, time.Now()); err != nil {
		return nil, err
	}

	writer := &dailyFile{
		basePath: basePath,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	writer.mu.Lock()
	err := writer.rotateLocked(time.Now())
	writer.mu.Unlock()
	if err != nil {
		if writer.file != nil {
			_ = writer.file.Close()
		}
		return nil, err
	}

	go writer.rotateAtMidnight()
	return writer, nil
}

func (w *dailyFile) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return 0, os.ErrClosed
	}
	if err := w.rotateLocked(time.Now()); err != nil {
		// Keep capturing errors in the current file if a new daily file cannot open.
		return w.file.Write(p)
	}
	return w.file.Write(p)
}

func (w *dailyFile) rotateAtMidnight() {
	defer close(w.done)
	for {
		now := time.Now()
		y, m, d := now.Date()
		nextMidnight := time.Date(y, m, d+1, 0, 0, 0, 0, now.Location())
		timer := time.NewTimer(time.Until(nextMidnight))
		select {
		case <-timer.C:
			w.mu.Lock()
			_ = w.rotateLocked(time.Now())
			w.mu.Unlock()
		case <-w.stop:
			timer.Stop()
			return
		}
	}
}

func (w *dailyFile) rotateLocked(now time.Time) error {
	day := now.Format(logDateLayout)
	if day == w.day && w.file != nil {
		return nil
	}

	file, err := os.OpenFile(datedPath(w.basePath, day), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	oldFile := w.file
	w.file = file
	w.day = day
	if oldFile != nil {
		_ = oldFile.Close()
	}
	return pruneOldFiles(w.basePath, now)
}

func (w *dailyFile) Close() error {
	w.stopOnce.Do(func() { close(w.stop) })
	<-w.done

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func datedPath(basePath, day string) string {
	extension := filepath.Ext(basePath)
	name := strings.TrimSuffix(filepath.Base(basePath), extension)
	return filepath.Join(filepath.Dir(basePath), name+"-"+day+extension)
}

func migrateLegacyFile(basePath string, now time.Time) error {
	info, err := os.Stat(basePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	legacyDay := info.ModTime().In(now.Location()).Format(logDateLayout)
	if olderThanRetention(legacyDay, now) {
		return os.Remove(basePath)
	}

	targetPath := datedPath(basePath, legacyDay)
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		if err := os.Rename(basePath, targetPath); err == nil {
			return nil
		}
	} else if err != nil {
		return err
	}

	source, err := os.Open(basePath)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Remove(basePath)
}

func pruneOldFiles(basePath string, now time.Time) error {
	dir := filepath.Dir(basePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		day, ok := datedFileDay(basePath, entry.Name())
		if !ok || !olderThanRetention(day, now) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func datedFileDay(basePath, filename string) (string, bool) {
	extension := filepath.Ext(basePath)
	baseName := strings.TrimSuffix(filepath.Base(basePath), extension)
	prefix := baseName + "-"
	if !strings.HasPrefix(filename, prefix) || filepath.Ext(filename) != extension {
		return "", false
	}
	day := strings.TrimSuffix(strings.TrimPrefix(filename, prefix), extension)
	parsed, err := time.ParseInLocation(logDateLayout, day, time.Local)
	return day, err == nil && parsed.Format(logDateLayout) == day
}

func olderThanRetention(day string, now time.Time) bool {
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -2)
	return day < cutoff.Format(logDateLayout)
}
