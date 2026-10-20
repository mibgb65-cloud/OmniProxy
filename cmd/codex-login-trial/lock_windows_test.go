package main

import (
	"testing"
	"time"
)

func TestTrialLockPreventsConcurrentClientChanges(t *testing.T) {
	home := t.TempDir()
	release, err := lockTrial(home)
	if err != nil {
		t.Fatal(err)
	}
	held := true
	defer func() {
		if held {
			release()
		}
	}()
	blocked := make(chan bool, 1)
	go func() {
		next, err := lockTrial(home)
		if err == nil {
			next()
		}
		blocked <- err != nil
	}()
	select {
	case refused := <-blocked:
		if !refused {
			t.Fatal("concurrent trial entered the same client directory")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent trial did not fail promptly")
	}
	release()
	held = false
	next, err := lockTrial(home)
	if err != nil {
		t.Fatal("trial lock not released")
	}
	next()
}
