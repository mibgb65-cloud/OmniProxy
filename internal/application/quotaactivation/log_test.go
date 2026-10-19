package quotaactivation

import (
	"bytes"
	"context"
	"omniproxy/internal/infrastructure/logger"
	"omniproxy/internal/platform/profile"
	"strings"
	"testing"
	"time"
)

func TestActivationLogsOnlySafeCounts(t *testing.T) {
	var output bytes.Buffer
	log, files, err := logger.Setup(logger.Options{Profile: profile.Test, Console: &output})
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	f := newFixture(t)
	f.enable(t)
	f.accounts.item.Name = "private-account-name"
	f.service.SetLogger(log)
	f.quotas.set(floatingQuota(time.Now().UTC().Truncate(time.Second)))
	if err = f.service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	s := output.String()
	if !strings.Contains(s, "quotaactivation") || !strings.Contains(s, "pending=1") || strings.Contains(s, "private-account-name") || strings.Contains(s, f.accounts.item.AuthEmail) || strings.Contains(s, f.accounts.item.AuthAccountID) {
		t.Fatal("activation diagnosis missing or identity leaked")
	}
}
