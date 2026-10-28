package netport

import (
	"errors"
	"net"
	"testing"
)

func TestAvailableRejectsOutOfRange(t *testing.T) {
	for _, port := range []int{0, -1, MaxPort + 1} {
		if Available(port) {
			t.Fatalf("Available(%d) = true, want false", port)
		}
	}
}

func TestNextAvailableSkipsOccupiedPort(t *testing.T) {
	occupied, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	port := occupied.Addr().(*net.TCPAddr).Port

	if Available(port) {
		t.Fatalf("port %d reported available while listened", port)
	}
	next, err := NextAvailable(port, port+32)
	if err != nil {
		t.Fatalf("NextAvailable error: %v", err)
	}
	if next <= port || next > port+32 {
		t.Fatalf("NextAvailable = %d, want within (%d, %d]", next, port, port+32)
	}
	if !Available(next) {
		t.Fatalf("chosen port %d is not available", next)
	}
}

func TestAvailableAcceptsFreePort(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if !Available(port) {
		t.Fatalf("port %d reported unavailable after release", port)
	}
}

func TestNextAvailableReportsExhaustedRange(t *testing.T) {
	occupied, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	port := occupied.Addr().(*net.TCPAddr).Port

	if port == MaxPort {
		t.Skip("allocated port is the maximum TCP port")
	}
	if _, err := NextAvailable(port, port); !errors.Is(err, ErrNoAvailablePort) {
		t.Fatalf("NextAvailable error = %v, want ErrNoAvailablePort", err)
	}
}
