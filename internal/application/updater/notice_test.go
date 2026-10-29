package updater

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/infrastructure/appinstall"
)

// 停止 fixture 的后台任务，避免临时目录在下载协程仍在写时被清理。
func closeFixture(t *testing.T, s *Service) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

// installFixture 走完一次真实的「发现 → 下载 → 安装启动」，返回安装前的服务与重启选项。
func installFixture(t *testing.T) (*Service, Options) {
	t.Helper()
	s, opts, launches := fixture(t, false)
	state, err := s.Check(context.Background())
	if err != nil || state.Release == nil {
		t.Fatalf("check=%+v %v", state, err)
	}
	if _, err := s.Start(state.Release.Version); err != nil {
		t.Fatal(err)
	}
	waitTransfer(t, s, "ready")
	if _, err := s.Install(state.Release.Version); err != nil {
		t.Fatal(err)
	}
	if *launches != 1 {
		t.Fatalf("launches=%d", *launches)
	}
	closeFixture(t, s)
	// 安装器替换程序后重启：同一更新目录，运行版本已经是刚安装的版本。
	// 运行版本不带 v 前缀，发布标签带 v 前缀，两者由 version.Compare 判定相等。
	restarted := opts
	restarted.Version = strings.TrimPrefix(state.Release.Version, "v")
	return s, restarted
}

func TestInstalledUpdateProducesNoticeOnce(t *testing.T) {
	_, restarted := installFixture(t)
	state := New(restarted).Snapshot()
	if state.Download.State != "installed" {
		t.Fatalf("download=%+v", state.Download)
	}
	if state.Installed == nil {
		t.Fatal("completed update produced no notice")
	}
	if state.Installed.Version != "2.0.0-beta.2" || state.Installed.From != "2.0.0-beta.1" {
		t.Fatalf("notice=%+v", *state.Installed)
	}
	service := New(restarted)
	if _, err := service.AcknowledgeInstalled(); err != nil {
		t.Fatal(err)
	}
	if acknowledged := service.Snapshot(); acknowledged.Installed != nil || acknowledged.CurrentVersion != "2.0.0-beta.2" {
		t.Fatalf("acknowledged=%+v", acknowledged)
	}
	if again, err := service.AcknowledgeInstalled(); err != nil || again.Installed != nil {
		t.Fatalf("second acknowledge=%+v %v", again, err)
	}
}

func TestAcknowledgedNoticeDoesNotReturn(t *testing.T) {
	_, restarted := installFixture(t)
	service := New(restarted)
	if _, err := service.AcknowledgeInstalled(); err != nil {
		t.Fatal(err)
	}
	// 同一目录再次启动时读取 status.json，不能因为「版本仍高于记录」重复提示。
	third := New(restarted).Snapshot()
	if third.Installed != nil {
		t.Fatal("acknowledged notice returned after restart")
	}
	if third.Download.State != "installed" {
		t.Fatalf("download=%+v", third.Download)
	}
}

func TestPendingInstallProducesNoNotice(t *testing.T) {
	s, opts, _ := fixture(t, false)
	state, err := s.Check(context.Background())
	if err != nil || state.Release == nil {
		t.Fatalf("check=%+v %v", state, err)
	}
	if _, err := s.Start(state.Release.Version); err != nil {
		t.Fatal(err)
	}
	waitTransfer(t, s, "ready")
	closeFixture(t, s)
	// 只下载未安装：运行版本仍低于记录版本，不能宣称升级完成。
	if notice := New(opts).Snapshot().Installed; notice != nil {
		t.Fatalf("notice=%+v", *notice)
	}
}

func TestFailedInstallReceiptProducesNoNotice(t *testing.T) {
	_, restarted := installFixture(t)
	receipt, _ := json.Marshal(appinstall.Receipt{Version: "v2.0.0-beta.2", Success: false, Code: "install: failed"})
	if err := os.WriteFile(filepath.Join(restarted.Directory, "install-result.json"), receipt, 0600); err != nil {
		t.Fatal(err)
	}
	state := New(restarted).Snapshot()
	if state.Download.State != "failed" || state.Installed != nil {
		t.Fatalf("snapshot=%+v", state)
	}
}
