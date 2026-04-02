package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	_ "github.com/daominah/reminderd/pkg/driver/base"
	"github.com/daominah/reminderd/pkg/driver/notify"
	"github.com/daominah/reminderd/pkg/driver/userinput"
	"github.com/daominah/reminderd/pkg/logic"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tracker := logic.NewUserInputTracker(
		&userinput.IdleDetector{},
		&notify.OSNotifier{},
	)
	if err := tracker.Run(ctx); err != nil {
		log.Fatalf("error tracker.Run: %v", err)
	}
}
