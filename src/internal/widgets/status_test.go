package widgets

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"
)

var tagRe = regexp.MustCompile(`\[[^\]]*\]`)

func plain(s string) string { return tagRe.ReplaceAllString(s, "") }

func TestSyncStatusRender(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.Local)
	s := newSyncStatus()

	if got := plain(s.Render(now)); !strings.Contains(got, "not set up") {
		t.Errorf("unconfigured: %q", got)
	}

	s.SetConfigured(true)
	if got := plain(s.Render(now)); !strings.Contains(got, "waiting for first sync") {
		t.Errorf("configured, no sync yet: %q", got)
	}

	s.Started()
	if got := plain(s.Render(now)); !strings.Contains(got, "syncing") {
		t.Errorf("syncing: %q", got)
	}

	s.Finished(nil, now.Add(-3*time.Minute))
	if got := plain(s.Render(now)); !strings.Contains(got, "synced 3m ago") {
		t.Errorf("synced: %q", got)
	}

	offline := fmt.Errorf("canvas request failed: %w", &net.DNSError{Err: "no such host", Name: "x.instructure.com"})
	s.Finished(offline, now)
	if got := plain(s.Render(now)); !strings.Contains(got, "offline") || !strings.Contains(got, "last synced 3m ago") {
		t.Errorf("offline: %q", got)
	}

	s.Finished(errors.New("canvas request returned 401 Unauthorized"), now)
	if got := plain(s.Render(now)); !strings.Contains(got, "sync failed") {
		t.Errorf("failed: %q", got)
	}
}
