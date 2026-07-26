package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/proxy"
	"omniproxy/internal/token"
	"strings"
	"time"
)

func (a *appServer) startProxy() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.proxyServer != nil {
		return nil
	}
	cfg := a.cfg

	server, listener, svc, err := a.newProxyServer(cfg)
	if err != nil {
		return err
	}

	a.proxyServer = server
	a.proxyService = svc

	a.serveProxy(server, listener)
	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: fmt.Sprintf("proxy started on port %d", cfg.ProxyPort)})
	return nil
}

// withLoopbackHost rejects requests whose Host header is not a loopback name.
// Both listeners bind 127.0.0.1, so any other Host means the request reached us
// through a rebound DNS name instead of a local client.
func withLoopbackHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHostHeader(r.Host) {
			http.Error(w, "host not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLoopbackHostHeader(hostHeader string) bool {
	host := strings.TrimSpace(hostHeader)
	if host == "" {
		return false
	}
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	return isLoopbackHost(host)
}

func (a *appServer) newProxyServer(cfg config.Config) (*http.Server, net.Listener, *proxy.Service, error) {
	if err := validateConfiguredPorts(cfg); err != nil {
		return nil, nil, nil, err
	}

	svc, err := proxy.NewService(cfg, a.tokens, a.logs, a.history)
	if err != nil {
		return nil, nil, nil, err
	}
	svc.SetTokenRefresher(a.refreshAuthTokenIfNeeded)
	if a.taskAutomation != nil {
		svc.SetActivityObserver(a.taskAutomation)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", cfg.ProxyPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("start proxy listener on %s: %w", addr, err)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           withLoopbackHost(svc),
		ReadHeaderTimeout: 30 * time.Second,
	}
	return server, listener, svc, nil
}

func (a *appServer) serveProxy(server *http.Server, listener net.Listener) {
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logs.Add(logs.Entry{Level: logs.LevelError, Message: fmt.Sprintf("proxy stopped: %v", err)})
			log.Printf("proxy stopped: %v", err)
		}
		a.mu.Lock()
		if a.proxyServer == server {
			a.proxyServer = nil
			a.proxyService = nil
		}
		a.mu.Unlock()
	}()
}

func (a *appServer) startControl() error {
	a.mu.Lock()
	if a.control != nil {
		a.mu.Unlock()
		return nil
	}
	cfg := a.cfg
	a.mu.Unlock()

	server, listener, err := a.newControlServer(cfg)
	if err != nil {
		return err
	}

	a.mu.Lock()
	if a.control != nil {
		a.mu.Unlock()
		_ = listener.Close()
		return nil
	}
	a.control = server
	a.mu.Unlock()

	a.serveControl(server, listener)
	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: fmt.Sprintf("control API started on port %d", cfg.ControlPort)})
	log.Printf("OmniProxy control API listening on http://%s", server.Addr)
	return nil
}

func (a *appServer) newControlServer(cfg config.Config) (*http.Server, net.Listener, error) {
	if err := validateConfiguredPorts(cfg); err != nil {
		return nil, nil, err
	}

	addr := fmt.Sprintf("127.0.0.1:%d", cfg.ControlPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("start control API listener on %s: %w", addr, err)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           withLoopbackHost(withCORS(a.routes())),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server, listener, nil
}

func (a *appServer) serveControl(server *http.Server, listener net.Listener) {
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logs.Add(logs.Entry{Level: logs.LevelError, Message: fmt.Sprintf("control API stopped: %v", err)})
			log.Printf("control API stopped: %v", err)
		}
		a.mu.Lock()
		if a.control == server {
			a.control = nil
		}
		a.mu.Unlock()
	}()
}

func (a *appServer) stopProxy() error {
	a.mu.Lock()
	server := a.proxyServer
	a.proxyServer = nil
	a.proxyService = nil
	a.mu.Unlock()
	if a.taskAutomation != nil {
		a.taskAutomation.Stop()
	}

	if server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := server.Shutdown(ctx)
	if err != nil {
		_ = server.Close()
	}
	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "proxy stopped"})
	return err
}

func (a *appServer) activeProxyRequests() []proxy.ActiveRequest {
	a.mu.Lock()
	svc := a.proxyService
	a.mu.Unlock()
	if svc == nil {
		return []proxy.ActiveRequest{}
	}
	return svc.ActiveRequests()
}

func (a *appServer) stopControl() error {
	a.mu.Lock()
	server := a.control
	a.control = nil
	a.mu.Unlock()

	if server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := server.Shutdown(ctx)
	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "control API stopped"})
	return err
}

func (a *appServer) restartProxy() error {
	a.mu.Lock()
	old := a.proxyServer
	cfg := a.cfg
	a.mu.Unlock()

	if old == nil {
		return a.startProxy()
	}

	addr := fmt.Sprintf("127.0.0.1:%d", cfg.ProxyPort)
	if old.Addr != addr {
		server, listener, svc, err := a.newProxyServer(cfg)
		if err != nil {
			return err
		}

		a.mu.Lock()
		if a.proxyServer != old {
			a.mu.Unlock()
			_ = listener.Close()
			return errors.New("proxy state changed during restart")
		}
		a.proxyServer = server
		a.proxyService = svc
		a.mu.Unlock()

		a.serveProxy(server, listener)
		a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: fmt.Sprintf("proxy started on port %d", cfg.ProxyPort)})
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = old.Shutdown(ctx)
		cancel()
		if err != nil {
			_ = old.Close()
			a.logs.Add(logs.Entry{Level: logs.LevelWarn, Message: fmt.Sprintf("proxy restart shutdown failed: %v", err)})
		} else {
			a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "proxy stopped"})
		}
		return nil
	}

	if err := a.stopProxy(); err != nil {
		return err
	}
	return a.startProxy()
}

func (a *appServer) restartControl() error {
	a.mu.Lock()
	old := a.control
	cfg := a.cfg
	a.mu.Unlock()

	if old == nil {
		return a.startControl()
	}

	server, listener, err := a.newControlServer(cfg)
	if err != nil {
		return err
	}

	a.activateControlServer(server, listener, cfg.ControlPort)
	return nil
}

func (a *appServer) activateControlServer(server *http.Server, listener net.Listener, port int) {
	a.mu.Lock()
	old := a.control
	a.control = server
	a.mu.Unlock()

	a.serveControl(server, listener)
	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: fmt.Sprintf("control API started on port %d", port)})
	log.Printf("OmniProxy control API listening on http://%s", server.Addr)

	if old != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := old.Shutdown(ctx); err != nil {
				a.logs.Add(logs.Entry{Level: logs.LevelWarn, Message: fmt.Sprintf("control API restart shutdown failed: %v", err)})
				return
			}
			a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "control API stopped"})
		}()
	}
}

func proxyConfigChanged(oldCfg config.Config, nextCfg config.Config) bool {
	return proxy.ProxyConfigChanged(oldCfg, nextCfg)
}

func controlConfigChanged(oldCfg config.Config, nextCfg config.Config) bool {
	return oldCfg.ControlPort != nextCfg.ControlPort
}

func maskedTokenValue(item token.Token) string {
	if strings.TrimSpace(item.TokenValue) == "" {
		return ""
	}
	if item.CredentialType == token.CredentialTypeCodexAuthJSON || item.CredentialType == token.CredentialTypeClaudeOAuth {
		return "auth.json"
	}
	value := strings.TrimSpace(item.TokenValue)
	if len(value) <= 12 {
		return value[:3] + "..."
	}
	return value[:7] + "..." + value[len(value)-4:]
}

func validateConfiguredPorts(cfg config.Config) error {
	if err := validateTCPPort("proxy port", cfg.ProxyPort); err != nil {
		return err
	}
	if err := validateTCPPort("control port", cfg.ControlPort); err != nil {
		return err
	}
	if cfg.ProxyPort == cfg.ControlPort {
		return errors.New("proxy port and control port must be different")
	}
	return nil
}

func validateTCPPort(name string, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535", name)
	}
	return nil
}

func isConfigValidationError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "port") ||
		strings.Contains(message, "url") ||
		strings.Contains(message, "invalid")
}
