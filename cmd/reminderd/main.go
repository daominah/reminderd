package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	_ "github.com/daominah/reminderd/pkg/driver/base"
	"github.com/daominah/reminderd/pkg/driver/config"
	"github.com/daominah/reminderd/pkg/driver/history"
	"github.com/daominah/reminderd/pkg/driver/httpsvr"
	"github.com/daominah/reminderd/pkg/driver/notify"
	"github.com/daominah/reminderd/pkg/driver/userinput"
	"github.com/daominah/reminderd/pkg/logic"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create data directory.
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error os.UserHomeDir: %v", err)
	}
	dataDir := filepath.Join(home, ".reminderd")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("error os.MkdirAll %s: %v", dataDir, err)
	}

	// Config.
	configStore := config.NewFileConfigStore(filepath.Join(dataDir, "config.json"))
	cfg, err := configStore.Load()
	if err != nil {
		log.Fatalf("error configStore.Load: %v", err)
	}

	// History.
	historyStore := history.NewFileStore(dataDir)

	// HTTP server.
	srv := httpsvr.NewServer(configStore, historyStore, cfg.WebUIPort)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("error httpsvr.ListenAndServe: %v", err)
		}
	}()

	// Tracker.
	tracker := logic.NewUserInputTracker(
		&userinput.IdleDetector{},
		&notify.OSNotifier{},
	)
	tracker.ConfigStore = configStore
	tracker.HistoryWriter = historyStore
	if err := tracker.Run(ctx); err != nil {
		log.Fatalf("error tracker.Run: %v", err)
	}
}
