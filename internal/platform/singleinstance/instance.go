// Package singleinstance guarantees a single OmniProxy process per runtime
// profile and lets a later launch bring the existing window to the front.
//
// The guard is taken before the configuration, the database, the migrations and
// the logger are touched: a duplicate launch must never read or modify shared
// state. The identity is the profile application ID, so production, development
// and test builds still run side by side (技术架构选择.md §24, PRD "single
// instance").
package singleinstance

import "errors"

// ErrAlreadyRunning reports that another process of the same profile holds the
// instance lock. The running instance has already been asked to show its window
// when this error is returned.
var ErrAlreadyRunning = errors.New("singleinstance: another instance is already running")

// Guard owns the process-unique lock and delivers activation requests.
type Guard interface {
	// OnActivate registers the callback that brings the existing window to the
	// front. A request that arrived before registration is delivered
	// immediately; later requests are delivered from a background goroutine.
	OnActivate(activate func())

	// Close releases the lock and stops delivering activation requests.
	Close() error
}

// Acquire takes the instance lock for the given profile application ID. It
// returns ErrAlreadyRunning when another process owns the lock.
func Acquire(uniqueID string) (Guard, error) { return acquire(uniqueID) }
