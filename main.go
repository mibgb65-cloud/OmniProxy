package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"omniproxy/internal/infrastructure/appinstall"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/infrastructure/logger"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/platform/startup"
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
	if handled, code := appinstall.HandleMode(os.Args[1:]); handled {
		os.Exit(code)
	}
	// Load .env (if present) before resolving the runtime profile so that
	// environment overrides take effect. Never log the file contents.
	_ = profile.LoadDotEnv(".env")

	prof, err := profile.Detect()
	if err != nil {
		slog.Error("failed to resolve runtime profile", "error", err)
		startup.NotifyFailure("无法读取运行配置", "")
		os.Exit(1)
	}
	if err := resolveDatabaseProfile(prof); err != nil {
		// 此阶段尚未建立常规日志；先记录并关闭，确保弹窗中的日志可读取。
		if log, files, logErr := logger.Setup(logger.Options{Profile: prof.Kind, LogDir: prof.LogDir}); logErr == nil {
			log.Error("failed to resolve database location", "error", err)
			_ = files.Close()
		} else {
			slog.Error("failed to resolve database location", "error", err, "log_error", logErr)
		}
		startup.NotifyFailure("无法识别现有数据库或保存新版数据位置", prof.LogDir)
		os.Exit(1)
	}

	log, logFiles, err := logger.Setup(logger.Options{
		Profile: prof.Kind,
		LogDir:  prof.LogDir,
	})
	if err != nil {
		slog.Error("failed to initialise logger", "error", err)
		startup.NotifyFailure("无法初始化日志", prof.LogDir)
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
		_ = logFiles.Close() // os.Exit 不运行 defer，失败提示前必须落盘。
		startup.NotifyFailure("无法打开数据库", prof.LogDir)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	migrationFiles, err := embeddedMigrations()
	if err != nil {
		log.Error("failed to open embedded migrations", "error", err)
		_ = logFiles.Close()
		startup.NotifyFailure("无法读取数据库迁移", prof.LogDir)
		os.Exit(1)
	}
	if err := database.Migrate(context.Background(), db, migrationFiles); err != nil {
		log.Error("failed to apply database migrations", "error", err)
		_ = logFiles.Close()
		startup.NotifyFailure("数据库迁移未完成", prof.LogDir)
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
		_ = logFiles.Close()
		startup.NotifyFailure("无法打开应用窗口", prof.LogDir)
		os.Exit(1)
	}
}

func resolveDatabaseProfile(prof *profile.Profile) error {
	path, err := database.ResolveLegacyPath(context.Background(), prof.DatabasePath())
	if err != nil || path == prof.DatabasePath() {
		return err
	}
	dir := filepath.Dir(path)
	if strings.TrimSpace(os.Getenv("OMNIPROXY_DATA_DIR")) == "" {
		cfg, err := profile.LoadConfig(prof.Kind)
		if err != nil {
			return err
		}
		// 这是隔离旧库，不是搬迁；不能设置 previousDataDir 触发搬迁回退。
		cfg.DataDir, cfg.PreviousDataDir = dir, ""
		if err := profile.SaveConfig(prof.Kind, cfg); err != nil {
			return err
		}
	}
	prof.DataDir, prof.LogDir = dir, filepath.Join(dir, "logs")
	return nil
}

func embeddedMigrations() (fs.FS, error) {
	return fs.Sub(migrations, "migrations")
}
