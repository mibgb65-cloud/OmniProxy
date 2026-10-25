// Package wails adapts OmniProxy application services to the Wails desktop
// shell. It is the ONLY package allowed to import the Wails runtime: the core
// must stay independent of Wails and React (技术架构选择.md §19, Rule 1-2).
package wails

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/agentconfig"
	"omniproxy/internal/application/agentinstall"
	"omniproxy/internal/application/gateway"
	"omniproxy/internal/application/proxysettings"
	"omniproxy/internal/application/quotaactivation"
	"omniproxy/internal/application/updater"
	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/infrastructure/appinstall"
	"omniproxy/internal/infrastructure/credential"
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
	gateway          *gateway.Service
	quotaActivation  *quotaactivation.Service
	agents           *agentconfig.Service
	installations    *agentinstall.Service
	activity         *activity.Service
	prof             *profile.Profile
	control          *httpapi.Server
	db               *sql.DB
	secrets          proxysettings.SecretStore
	proxySettings    *proxysettings.Service
	logFiles         *logger.Files
	log              *slog.Logger
	startedAt        time.Time
	settingsMu       sync.Mutex
	autostart        autostartController
	clipboardMu      sync.Mutex
	copyText         func(string) bool
	logActions       sync.Mutex
	selectLogExport  func(string) (string, error)
	openLogDirectory func(string) error
	openAgentURL     func(string) error
	stopLogs         func()
	stopActivity     func()
	subscription     subscriptionState
	googleAuth       googleSubscriptionState
	updatesMu        sync.Mutex
	updates          *updater.Service
	stopUpdateChecks context.CancelFunc
	quitForUpdate    func()
	activationMu     sync.Mutex
	emitActivation   func(quotaactivation.Status)
}

// NewAppService creates the bound application service.
func NewAppService(
	prof *profile.Profile,
	control *httpapi.Server,
	db *sql.DB,
	logFiles *logger.Files,
	log *slog.Logger,
) *AppService {
	service := &AppService{
		prof:      prof,
		control:   control,
		db:        db,
		logFiles:  logFiles,
		log:       log,
		startedAt: time.Now(),
	}
	service.initOutboundProxy()
	service.initGateway()
	service.initAgentInstaller()
	return service
}

// initOutboundProxy 安装出站代理解析器，让所有上游适配器按设置分流。
// 凭据库不可用时保持直连行为，不阻断应用启动。
func (s *AppService) initOutboundProxy() {
	if s.db == nil || s.prof == nil {
		return
	}
	store, err := credential.New(s.prof)
	if err != nil {
		if s.log != nil {
			s.log.Warn("outbound proxy credentials unavailable; provider proxy disabled", "module", "netproxy", "error", err)
		}
		return
	}
	s.secrets = store
	// 绑定层与出站分流使用同一个服务实例，保存后的失效才对两者同时生效。
	s.proxySettings = proxysettings.NewService(s.db, store)
	netproxy.Install(s.proxySettings)
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
	s.startUpdateChecks()
	s.startGoogleModelSync()
	if s.quotaActivation != nil {
		s.quotaActivation.Start(context.Background())
	}
	s.ensureBackgroundActivation()
	if err := s.control.Start(); err != nil {
		s.log.Error("control API failed to start; external clients cannot connect",
			"error", err, "control_port", s.prof.ControlPort)
	}
	if err := s.startGatewayOnStartup(); err != nil {
		s.log.Error("gateway failed to start automatically; manual retry is available",
			"module", "gateway", "error", err)
	}
	return nil
}

// ServiceShutdown stops the local Control API and relocates the data
// directory when the user changed it.
func (s *AppService) ServiceShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if s.stopUpdateChecks != nil {
		s.stopUpdateChecks()
	}
	s.updatesMu.Lock()
	updates := s.updates
	s.updatesMu.Unlock()
	var updateErr error
	if updates != nil {
		updateErr = updates.Close(ctx)
	}
	if s.quotaActivation != nil {
		s.quotaActivation.Stop()
	}
	if s.stopActivity != nil {
		s.stopActivity()
	}
	s.shutdownSubscription()
	s.shutdownGoogleSubscription()
	s.shutdownLogs()
	var installationErr error
	if s.installations != nil {
		installationErr = s.installations.Close(ctx)
	}
	gatewayErr := s.shutdownGateway(ctx)
	var activationErr error
	if s.quotaActivation != nil {
		activationErr = s.quotaActivation.Close(ctx)
	}
	if err := s.control.Shutdown(ctx); err != nil {
		s.log.Error("control API shutdown failed", "error", err)
	}
	s.relocateIfPending()
	return errors.Join(gatewayErr, installationErr, activationErr, updateErr)
}

// Run creates the Wails application, opens the main window and blocks until
// the application exits.
func Run(app *App) error {
	if err := waitForFrontend(context.Background(), app.Service.log); err != nil {
		return err
	}

	wailsApp := application.New(application.Options{
		Name:        "OmniProxy",
		Description: "Local AI API gateway and multi-account management tool",
		Icon:        appIcon,
		Services:    []application.Service{application.NewService(app.Service)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(app.Assets)},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	app.Service.quitForUpdate = wailsApp.Quit
	stopInstallerListener, listenerErr := appinstall.ListenForQuit(wailsApp.Quit)
	if listenerErr != nil {
		app.Service.log.Warn("installer quit listener unavailable", "error", listenerErr)
	} else {
		defer func() {
			if err := stopInstallerListener(); err != nil {
				app.Service.log.Warn("installer quit listener close failed", "error", err)
			}
		}()
	}

	app.Service.bindGatewayEvents(wailsApp)
	app.Service.bindActivityEvents(wailsApp)
	app.Service.autostart = wailsApp.Autostart
	app.Service.copyText = wailsApp.Clipboard.SetText
	app.Service.openAgentURL = wailsApp.Browser.OpenURL
	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "OmniProxy",
		Frameless:        true,
		Width:            1200,
		Height:           800,
		MinWidth:         960,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(9, 9, 11),
		URL:              "/",
	})
	app.Service.bindLogEvents(wailsApp, window)
	app.Service.bindSubscriptionEvents(wailsApp)
	app.Service.bindActivationEvents(wailsApp)
	tray := wailsApp.SystemTray.New()
	tray.SetIcon(appIcon)
	tray.SetTooltip("OmniProxy")
	menu := wailsApp.NewMenu()
	menu.Add("显示 OmniProxy / Show").OnClick(func(_ *application.Context) {
		window.Show().Focus()
	})
	menu.AddSeparator()
	menu.Add("退出 OmniProxy / Quit").OnClick(func(_ *application.Context) {
		wailsApp.Quit()
	})
	tray.SetMenu(menu)
	tray.OnClick(func() { window.Show().Focus() })

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if app.Service.shouldCloseToTray() {
			window.Hide()
			event.Cancel()
		}
	})

	return wailsApp.Run()
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}
