package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
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

// monitoredFiles keeps track of files and their stability
var monitoredFiles = make(map[string]fileInfo)

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

Loop: // Label for breaking out of the loop
	for {
		scanFolder(ctx) // Scan the folder for changes

		select {
		case <-time.After(scanInterval):
			// Continue to next scan
		case <-ctx.Done():
			break Loop
		}
	}
	slog.InfoContext(ctx, "Folder watcher stopped.")
}

func scanFolder(ctx context.Context) {
	start := time.Now()

	// Open the directory to use its ReadDir method
	dir, err := os.Open(watchFolder)
	if err != nil {
		slog.WarnContext(ctx, "Error opening directory", "folder", watchFolder, "error", err)
		return
	}
	defer dir.Close()

	// Loop to read directory entries in batches of 10
	for {
		dirEntries, err := dir.ReadDir(10) // Read up to 10 entries in a batch
		switch {
		case errors.Is(err, io.EOF):
			return
		case !errors.Is(err, nil):
			slog.WarnContext(ctx, "Error reading directory batch with File.ReadDir method", "folder", watchFolder, "error", err)
			return
		}

		// Process the current batch of entries
		for _, entry := range dirEntries { // entry is now os.DirEntry
			// Check for cancellation before processing each entry
			select {
			case <-ctx.Done():
				return
			default:
			}

			now := time.Now()

			if start.Add(scanInterval).Before(now) {
				slog.WarnContext(ctx, "Scan interval exceeded, stopping scan", "interval", scanInterval)
				return
			}

			if entry.IsDir() {
				continue
			}

			path := filepath.Join(watchFolder, entry.Name())
			stat, err := entry.Info()
			if err != nil {
				slog.WarnContext(ctx, "Error getting info for file entry", "path", path, "error", err)
				continue
			}

			size := stat.Size()
			previous, exists := monitoredFiles[path]
			if !exists || size != previous.size {
				monitoredFiles[path] = fileInfo{
					size:     size,
					lastSeen: now,
				}
				continue // new file or size changed, continue to next entry
			}

			if previous.lastSeen.Add(stableInterval).After(now) {
				continue
			}

			if err := process(path); err != nil {
				slog.ErrorContext(ctx, "Failed to process file", "path", path, "error", err)
				continue
			}

			if err := os.Remove(path); err != nil {
				slog.ErrorContext(ctx, "Failed to remove file after processing", "path", path, "error", err)
				continue
			}

			delete(monitoredFiles, path)

			slog.InfoContext(ctx, "File processed and removed", "path", path, "size", size)
		}
	}
}

func process(_ string) error {
	time.Sleep(1 * time.Second)
	return nil
}
