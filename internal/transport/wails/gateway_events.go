package wails

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"omniproxy/internal/core/proxy"
)

func init() { application.RegisterEvent[proxy.Status]("gateway:status") }

func (s *AppService) bindGatewayEvents(app *application.App) {
	if s.gateway != nil {
		s.gateway.OnStatus(func(status proxy.Status) { app.Event.Emit("gateway:status", status) })
	}
}
