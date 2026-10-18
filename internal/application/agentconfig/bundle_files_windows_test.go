//go:build windows

package agentconfig

import (
	"context"
	"testing"

	"golang.org/x/sys/windows"
)

func TestBundleSecondFileLockRollsBackFirstWrite(t *testing.T) {
	s, _ := gatewayClientFixture(t)
	path := s.paths.Pi
	writeGatewayFixture(t, path, gatewayAgentOriginal("pi"))
	settings := bundlePaths("pi", path)[1]
	writeGatewayFixture(t, settings, `{"theme":"dark"}`)
	before, err := s.snapshotClient("pi", path)
	if err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(settings)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(lock)
	if _, err := s.Apply(context.Background(), Input{Client: "pi", Revision: before.revision}); err == nil {
		t.Fatal("locked second file accepted")
	}
	after, err := s.snapshotClient("pi", path)
	if err != nil || after.revision != before.revision {
		t.Fatal("second file failure did not roll back first file", err)
	}
}
