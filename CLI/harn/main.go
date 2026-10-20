package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AwayerCode/Harn/CLI/harn/internal/app"
	"github.com/AwayerCode/Harn/CLI/harn/internal/buildinfo"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runner := app.New(app.Options{
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Version: buildinfo.Version,
		Client:  &http.Client{Timeout: 5 * time.Minute},
	})
	os.Exit(runner.Run(ctx, os.Args[1:]))
}
