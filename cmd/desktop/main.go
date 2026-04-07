//go:build desktop

package main

import (
	"context"
	"log/slog"
	"os"

	"md/internal/desktop"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

var (
	Version = "dev"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	runtime, err := desktop.Start(Version)
	if err != nil {
		slog.Error("failed to start desktop runtime", "error", err)
		os.Exit(1)
	}

	err = wails.Run(&options.App{
		Title:     "MD",
		Width:     1440,
		Height:    920,
		MinWidth:  1100,
		MinHeight: 680,
		AssetServer: &assetserver.Options{
			Handler: runtime.Handler(),
		},
		Mac:     &mac.Options{},
		Windows: &windows.Options{},
		OnShutdown: func(_ context.Context) {
			runtime.Stop()
		},
	})
	if err != nil {
		slog.Error("desktop app exited with error", "error", err)
		runtime.Stop()
		os.Exit(1)
	}
}
