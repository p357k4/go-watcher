package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	watchFolder    = "incoming"       // The folder to monitor
	scanInterval   = 5 * time.Second  // How often to scan the folder
	stableInterval = 30 * time.Second // Time to wait for file stability before processing
	logFile        = "app.log"        // Log file name
)

// fileInfo stores information about a file being monitored
type fileInfo struct {
	size     int64
	lastSeen time.Time
}

func main() {
	// Setup slog
	logFileHandle, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v\n", logFile, err)
		os.Exit(1)
	}
	defer logFileHandle.Close()

	multiWriter := io.MultiWriter(os.Stdout, logFileHandle)
	logger := slog.New(slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level:     slog.LevelInfo, // Set desired log level
		AddSource: true,           // Include source file and line number
	}))
	slog.SetDefault(logger)

	// Use signal.NotifyContext for simpler signal handling
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop() // Release resources associated with signal notification

	// Always attempt to create the watchFolder
	if err := os.MkdirAll(watchFolder, 0755); err != nil {
		slog.ErrorContext(ctx, "Error creating/accessing folder", "folder", watchFolder, "error", err)
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Watcher started. Press Ctrl+C to exit.")

	m := monitor{}

Loop: // Label for breaking out of the loop
	for {
		select {
		case <-time.After(scanInterval):
			// Continue to next scan
		case <-ctx.Done():
			break Loop
		}

		if err := fs.WalkDir(&myfs{}, watchFolder, m.monitor); err != nil {
			slog.ErrorContext(ctx, "Error processing files", "folder", watchFolder, "error", err)
			continue
		}
	}

	slog.InfoContext(ctx, "Folder watcher stopped.")
}

type myfs struct {
}

// ReadDir implements fs.ReadDirFS.
func (m *myfs) ReadDir(name string) ([]fs.DirEntry, error) {
	dir, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// Open implements fs.FS.
func (m *myfs) Open(name string) (fs.File, error) {
	return os.Open(name)
}

var _ fs.FS = (*myfs)(nil)
var _ fs.ReadDirFS = (*myfs)(nil)

type monitor struct {
	// monitoredFiles keeps track of files and their stability
	monitored map[string]fileInfo
}

func (m *monitor) monitor(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}

	now := time.Now()

	info, err := d.Info()
	if err != nil {
		return err
	}

	size := info.Size()
	previous, exists := m.monitored[path]
	if !exists || size != previous.size {
		m.monitored[path] = fileInfo{
			size:     size,
			lastSeen: now,
		}
		return nil
	}

	if previous.lastSeen.Add(stableInterval).After(now) {
		return nil
	}

	time.Sleep(1 * time.Second) // Simulate processing delay

	if err := os.Remove(path); err != nil {
		return nil
	}

	delete(m.monitored, path)

	return nil
}
