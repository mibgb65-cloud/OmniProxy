//go:build windows

package singleinstance

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// testID scopes lock names to one test so runs never collide with a real
// OmniProxy instance or with another test.
func testID(t *testing.T) string {
	t.Helper()
	return "test.omniproxy." + strings.ReplaceAll(t.Name(), "/", ".")
}

func mustAcquire(t *testing.T, id string) Guard {
	t.Helper()
	instance, err := Acquire(id)
	if err != nil {
		t.Fatalf("Acquire(%q) = %v, want nil", id, err)
	}
	t.Cleanup(func() { _ = instance.Close() })
	return instance
}

func TestSecondInstanceActivatesTheRunningOne(t *testing.T) {
	id := testID(t)
	first := mustAcquire(t, id)

	activated := make(chan struct{}, 1)
	first.OnActivate(func() {
		select {
		case activated <- struct{}{}:
		default:
		}
	})

	second, err := Acquire(id)
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second Acquire(%q) = %v, want ErrAlreadyRunning", id, err)
	}
	if second != nil {
		t.Fatalf("second Acquire returned a guard, want nil")
	}

	select {
	case <-activated:
	case <-time.After(3 * time.Second):
		t.Fatal("running instance was not asked to show its window")
	}
}

func TestActivationRequestedBeforeRegistrationIsDelivered(t *testing.T) {
	id := testID(t)
	instance := mustAcquire(t, id)
	concrete, ok := instance.(*guard)
	if !ok {
		t.Fatalf("Acquire(%q) returned %T, want *guard", id, instance)
	}

	// 窗口尚未创建时收到请求：先记住，注册回调时立即补发。
	concrete.fire()

	activated := make(chan struct{}, 1)
	instance.OnActivate(func() {
		select {
		case activated <- struct{}{}:
		default:
		}
	})

	select {
	case <-activated:
	default:
		t.Fatal("pending activation was not delivered during registration")
	}
}

func TestInstanceLockIsSeparatePerProfile(t *testing.T) {
	production := testID(t) + ".com.omniproxy.app"
	development := testID(t) + ".com.omniproxy.app.dev"

	productionGuard := mustAcquire(t, production)
	developmentGuard := mustAcquire(t, development)

	productionActivated := make(chan struct{}, 1)
	developmentActivated := make(chan struct{}, 1)
	productionGuard.OnActivate(func() {
		select {
		case productionActivated <- struct{}{}:
		default:
		}
	})
	developmentGuard.OnActivate(func() {
		select {
		case developmentActivated <- struct{}{}:
		default:
		}
	})

	if _, err := Acquire(production); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second Acquire(%q) = %v, want ErrAlreadyRunning", production, err)
	}

	select {
	case <-productionActivated:
	case <-time.After(3 * time.Second):
		t.Fatal("production instance was not asked to show its window")
	}
	select {
	case <-developmentActivated:
		t.Fatal("activation request leaked to another profile")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestCloseReleasesInstanceLock(t *testing.T) {
	id := testID(t)
	first, err := Acquire(id)
	if err != nil {
		t.Fatalf("Acquire(%q) = %v, want nil", id, err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close() = %v, want nil", err)
	}
	second, err := Acquire(id)
	if err != nil {
		t.Fatalf("Acquire(%q) after Close = %v, want nil", id, err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("Close() = %v, want nil", err)
	}
}

func TestAcquireRejectsEmptyID(t *testing.T) {
	guard, err := Acquire("")
	if err == nil || errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("Acquire(\"\") = (%v, %v), want a configuration error", guard, err)
	}
}

func TestInstanceNamesArePerProfile(t *testing.T) {
	if mutexName("com.omniproxy.app") == mutexName("com.omniproxy.app.dev") {
		t.Fatal("mutex names must differ between profiles")
	}
	if eventName("com.omniproxy.app") == eventName("com.omniproxy.app.dev") {
		t.Fatal("activation event names must differ between profiles")
	}
	if mutexName("com.omniproxy.app") == eventName("com.omniproxy.app") {
		t.Fatal("mutex and activation event must not share a name")
	}
	if !strings.HasSuffix(mutexName("com.omniproxy.app"), "com.omniproxy.app") {
		t.Fatal("mutex name must identify the profile application ID")
	}
}
