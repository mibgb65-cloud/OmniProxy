package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"os"

	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/infrastructure/logger"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/transport/httpapi"
	wailsapp "omniproxy/internal/transport/wails"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
//
//go:embed all:frontend/dist
var assets embed.FS

// migrations embeds the SQL migration files. Database schema changes must only
// be applied through these migrations (see docs/architecture/OVERVIEW.md).
//
//go:embed all:migrations
var migrations embed.FS

func main() {
	// Load .env (if present) before resolving the runtime profile so that
	// environment overrides take effect. Never log the file contents.
	_ = profile.LoadDotEnv(".env")

	prof, err := profile.Detect()
	if err != nil {
		slog.Error("failed to resolve runtime profile", "error", err)
		os.Exit(1)
	}

	log, logFiles, err := logger.Setup(logger.Options{
		Profile: prof.Kind,
		LogDir:  prof.LogDir,
	})
	if err != nil {
		slog.Error("failed to initialise logger", "error", err)
		os.Exit(1)
	}
	defer func() { _ = logFiles.Close() }()

	log.Info("starting OmniProxy",
		"profile", prof.Kind,
		"app_id", prof.AppID,
		"data_dir", prof.DataDir,
		"proxy_port", prof.ProxyPort,
		"control_port", prof.ControlPort,
	)

	db, err := database.Open(database.Config{Path: prof.DatabasePath()})
	if err != nil {
		log.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	migrationFiles, err := embeddedMigrations()
	if err != nil {
		log.Error("failed to open embedded migrations", "error", err)
		os.Exit(1)
	}
	if err := database.Migrate(context.Background(), db, migrationFiles); err != nil {
		log.Error("failed to apply database migrations", "error", err)
		os.Exit(1)
	}

	control := httpapi.New(prof)
	service := wailsapp.NewAppService(prof, control, db, logFiles, log)

	app := &wailsapp.App{
		Profile:  prof,
		Service:  service,
		Assets:   assets,
		DB:       db,
		LogFiles: logFiles,
	}
	if err := wailsapp.Run(app); err != nil {
		log.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}

func embeddedMigrations() (fs.FS, error) {
	return fs.Sub(migrations, "migrations")
}
