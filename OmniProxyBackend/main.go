package main

import (
	"context"
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"log"
	"net/http"
	"omniproxy/internal/config"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/proxy"
	"omniproxy/internal/taskautomation"
	"omniproxy/internal/token"
	"os"
	"strings"
	"sync"
	"time"
)

//go:embed all:frontend-dist
var assets embed.FS

type appServer struct {
	mu                    sync.Mutex
	configMu              sync.Mutex
	codexRefreshMu        sync.Mutex
	codexActivationMu     sync.Mutex
	codexOAuthMu          sync.Mutex
	codexOAuthSession     *codexOAuthSession
	claudeOAuthMu         sync.Mutex
	claudeOAuthSession    *claudeOAuthSession
	codexResetCreditMu    sync.Mutex
	dataDir               string
	cfg                   config.Config
	configStore           *config.Store
	tokens                *token.Manager
	logs                  *logs.Recorder
	history               *history.Recorder
	proxyServer           *http.Server
	proxyService          *proxy.Service
	premProxy             *premProxyManager
	taskAutomation        *taskautomation.Manager
	control               *http.Server
	controlToken          string
	updates               *updateDownloader
	healthStop            context.CancelFunc
	openRouterModelsMu    sync.Mutex
	openRouterModelsCache openRouterModelsCache
}

const (
	healthCheckTick       = time.Minute
	activeHealthInterval  = 15 * time.Minute
	retryHealthInterval   = time.Minute
	currentQuotaInterval  = 30 * time.Second
	healthRequestTimeout  = 15 * time.Second
	failedHealthRetryWait = 5 * time.Minute
	controlTokenHeader    = "X-OmniProxy-Control-Token"
	requestHistoryMax     = 50000
	defaultHistoryLimit   = 10000
)

const (
	historyEventManualValidation  = "manual-validation"
	historyEventCodexRefreshAdd   = "codex-refresh-after-add"
	historyEventStartupCodexUsage = "startup-codex-usage-refresh"
	historyEventCurrentQuota      = "current-quota-refresh"
	historyEventHealthCheck       = "health-check"
)

func main() {
	config.SetRuntimeProfile(appRuntimeMode())

	server, err := newAppServer()
	if err != nil {
		log.Fatalf("initialise app: %v", err)
	}

	desktop := NewDesktopApp(server)

	err = wails.Run(&options.App{
		Title:             appDisplayName(),
		Width:             1280,
		Height:            860,
		MinWidth:          1240,
		MinHeight:         720,
		Frameless:         true,
		StartHidden:       startHiddenFromArgs(),
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 238, G: 242, B: 247, A: 1},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               singleInstanceUniqueID(),
			OnSecondInstanceLaunch: desktop.secondInstanceLaunch,
		},
		OnStartup:  desktop.startup,
		OnShutdown: desktop.shutdown,
		Bind: []interface{}{
			desktop,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func startHiddenFromArgs() bool {
	for _, arg := range os.Args[1:] {
		switch strings.ToLower(strings.TrimSpace(arg)) {
		case "--minimized", "--hidden", "/minimized", "/hidden":
			return true
		}
	}
	return false
}
