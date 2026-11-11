//go:build !windows

package singleinstance

// acquire returns a guard without a platform lock. Single-instance enforcement
// is a Windows desktop requirement (PRD: Windows desktop UI, system tray,
// single instance); the other platforms only compile for tests and tooling.
func acquire(string) (Guard, error) { return noopGuard{}, nil }

type noopGuard struct{}

func (noopGuard) OnActivate(func()) {}

func (noopGuard) Close() error { return nil }
