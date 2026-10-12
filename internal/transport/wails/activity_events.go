package wails

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"sync"
	"time"
)

func init() { application.RegisterEvent[string]("activity:updated") }

func (s *AppService) bindActivityEvents(app *application.App) {
	if s.activity == nil {
		return
	}
	s.stopActivity = forwardActivityChanges(s.activity.Updates(), func() { app.Event.Emit("activity:updated", "") })
}

func forwardActivityChanges(updates <-chan struct{}, emit func()) func() {
	stop, done := make(chan struct{}), make(chan struct{})
	var once sync.Once
	go func() {
		defer close(done)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		dirty := false
		for {
			select {
			case <-stop:
				return
			case _, open := <-updates:
				if !open {
					return
				}
				dirty = true
			case <-ticker.C:
				if dirty {
					dirty = false
					emit()
				}
			}
		}
	}()
	return func() { once.Do(func() { close(stop); <-done }) }
}
