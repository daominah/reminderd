package main

import (
	"context"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/daominah/reminderd/pkg/base"
	"github.com/daominah/reminderd/pkg/driver/config"
	"github.com/daominah/reminderd/pkg/driver/history"
	"github.com/daominah/reminderd/pkg/driver/httpsvr"
	"github.com/daominah/reminderd/pkg/driver/notify"
	"github.com/daominah/reminderd/pkg/driver/userinput"
	"github.com/daominah/reminderd/pkg/logic"
	"github.com/daominah/reminderd/web"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error os.UserHomeDir: %v", err)
	}
	dataDir := filepath.Join(home, ".reminderd")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("error os.MkdirAll %s: %v", dataDir, err)
	}

	configStore := config.NewFileConfigStore(filepath.Join(dataDir, "config.json"))
	cfg, err := configStore.Load()
	if err != nil {
		log.Fatalf("error configStore.Load: %v", err)
	}

	historyStore := history.NewFileStore(dataDir)
	defer historyStore.Close()

	// Frontend: serve from disk if web/ dir exists (dev mode), otherwise use embedded files.
	var frontendFS fs.FS
	if rootDir, err := base.GetProjectRootDir(); err == nil {
		webDir := filepath.Join(rootDir, "web")
		if info, err := os.Stat(webDir); err == nil && info.IsDir() {
			log.Printf("dev mode: serving frontend from %s", webDir)
			frontendFS = os.DirFS(webDir)
		}
	}
	if frontendFS == nil {
		frontendFS = web.FrontendAssets
	}

	notifier := notify.ToastNotifier{}

	tracker := logic.NewUserInputTracker(
		&userinput.IdleDetector{},
		notifier,
	)
	tracker.ConfigStore = configStore
	tracker.HistoryWriter = historyStore
	tracker.HistoryReader = historyStore

	srv := httpsvr.NewServer(configStore, historyStore, frontendFS, cfg.WebUIPort)
	srv.Tracker = tracker
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("error httpsvr.ListenAndServe: %v", err)
		}
	}()

	if err := tracker.Run(ctx); err != nil {
		log.Fatalf("error tracker.Run: %v", err)
	}
}
