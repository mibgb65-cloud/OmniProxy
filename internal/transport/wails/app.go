// Package wails adapts OmniProxy application services to the Wails desktop
// shell. It is the ONLY package allowed to import the Wails runtime: the core
// must stay independent of Wails and React (技术架构选择.md §19, Rule 1-2).
package wails

import (
	"context"
	"database/sql"
	"embed"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"omniproxy/internal/infrastructure/logger"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/transport/httpapi"
	"omniproxy/internal/version"
)

// shutdownTimeout bounds the Control API graceful shutdown.
const shutdownTimeout = 5 * time.Second

// App bundles everything the desktop shell needs to run OmniProxy.
type App struct {
	Profile  *profile.Profile
	Service  *AppService
	Assets   embed.FS
	DB       *sql.DB
	LogFiles *logger.Files
}

// AppService exposes application information to the desktop UI through Wails
// typed bindings. It carries no business logic of its own: feature services
// live in internal/application/<feature> and are shared with the Control API.
type AppService struct {
	prof      *profile.Profile
	control   *httpapi.Server
	db        *sql.DB
	logFiles  *logger.Files
	log       *slog.Logger
	startedAt time.Time
}

// NewAppService creates the bound application service.
func NewAppService(
	prof *profile.Profile,
	control *httpapi.Server,
	db *sql.DB,
	logFiles *logger.Files,
	log *slog.Logger,
) *AppService {
	return &AppService{
		prof:      prof,
		control:   control,
		db:        db,
		logFiles:  logFiles,
		log:       log,
		startedAt: time.Now(),
	}
}

// AppInfo is a snapshot of application runtime information for the UI.
type AppInfo struct {
	Version     string `json:"version"`
	Profile     string `json:"profile"`
	AppID       string `json:"appId"`
	DataDir     string `json:"dataDir"`
	LogDir      string `json:"logDir"`
	ProxyPort   int    `json:"proxyPort"`
	ControlPort int    `json:"controlPort"`
}

// AppInfo returns the current application runtime information.
func (s *AppService) AppInfo() AppInfo {
	return AppInfo{
		Version:     version.Version,
		Profile:     string(s.prof.Kind),
		AppID:       s.prof.AppID,
		DataDir:     s.prof.DataDir,
		LogDir:      s.prof.LogDir,
		ProxyPort:   s.prof.ProxyPort,
		ControlPort: s.prof.ControlPort,
	}
}

// Ping is a liveness probe for the UI binding layer.
func (s *AppService) Ping() string { return "pong" }

// ServiceStartup starts the local Control API when the application starts.
// A bind failure (e.g. the port is already taken by another application) is
// logged loudly but does not abort the desktop app: the UI keeps working and
// the user can resolve the conflict. Returning an error here would make Wails
// terminate the whole application.
func (s *AppService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	if err := s.control.Start(); err != nil {
		s.log.Error("control API failed to start; external clients cannot connect",
			"error", err, "control_port", s.prof.ControlPort)
	}
	return nil
}

// ServiceShutdown stops the local Control API and relocates the data
// directory when the user changed it.
func (s *AppService) ServiceShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := s.control.Shutdown(ctx); err != nil {
		s.log.Error("control API shutdown failed", "error", err)
	}
	s.relocateIfPending()
	return nil
}

// Run creates the Wails application, opens the main window and blocks until
// the application exits.
func Run(app *App) error {
	wailsApp := application.New(application.Options{
		Name:        "OmniProxy",
		Description: "Local AI API gateway and multi-account management tool",
		Services:    []application.Service{application.NewService(app.Service)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(app.Assets)},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "OmniProxy",
		Frameless:        true,
		Width:            1200,
		Height:           800,
		MinWidth:         960,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(9, 9, 11),
		URL:              "/",
	})

	return wailsApp.Run()
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}
