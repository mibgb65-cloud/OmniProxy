//go:build windows

package codex

import (
	"runtime"
	"runtime/debug"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// 真实官方工作因子专项，使用假授权；不进入默认单测的 GiB 内存预算。
func BenchmarkEncryptedAuthOfficialFactor20(b *testing.B) {
	home := b.TempDir()
	b.Cleanup(func() { officialAuthCache.clearHome(home) })
	writeEncryptedAuthFixture(b, home, `{"tokens":{"access_token":"synthetic-only","account_id":"synthetic"}}`, 20)
	runtime.GC()
	debug.FreeOSMemory()
	for _, name := range []string{"cold", "cached"} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if name == "cold" {
					officialAuthCache.clearHome(home)
				}
				data, err := readEncryptedAuth(home, []byte("test-only-passphrase"))
				clear(data)
				if err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			b.ReportMetric(float64(mem.HeapAlloc)/(1<<20), "live_heap_MiB")
			b.ReportMetric(benchmarkPrivateBytes(b)/(1<<20), "private_MiB")
			if mem.HeapAlloc > 64<<20 {
				b.Fatal("finished auth read retained its GiB KDF workspace")
			}
		})
	}
}

func benchmarkPrivateBytes(b *testing.B) float64 {
	b.Helper()
	var counters struct {
		size, faults                                   uint32
		peakWS, ws, peakPaged, paged                   uintptr
		peakNonPaged, nonPaged, pagefile, peakPagefile uintptr
		private                                        uintptr
	}
	counters.size = uint32(unsafe.Sizeof(counters))
	ok, _, err := windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo").Call(
		uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&counters)), uintptr(counters.size))
	if ok == 0 {
		b.Fatal("process memory counters unavailable", err)
	}
	return float64(counters.private)
}
