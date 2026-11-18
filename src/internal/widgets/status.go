package widgets

import (
	"VIO/internal/dates"
	"VIO/internal/theme"
	"errors"
	"net"
	"time"

	"github.com/rivo/tview"
)

// SyncStatus is the one-line Canvas sync indicator in the dashboard header:
// "not set up", "syncing…", "synced 3m ago", "offline" or "sync failed".
// Call its methods on the UI goroutine (inside app.QueueUpdateDraw when
// coming from a background goroutine).
type SyncStatus struct {
	view       *tview.TextView
	configured bool
	syncing    bool
	lastOK     time.Time
	lastErr    error
}

func newSyncStatus() *SyncStatus {
	s := &SyncStatus{
		view: tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignCenter),
	}
	s.Refresh()
	return s
}

// SetConfigured records whether Canvas credentials are saved.
func (s *SyncStatus) SetConfigured(ok bool) {
	s.configured = ok
	if !ok {
		s.syncing = false
		s.lastErr = nil
	}
	s.Refresh()
}

// Started marks a sync as in progress.
func (s *SyncStatus) Started() {
	s.syncing = true
	s.Refresh()
}

// Finished records a sync result; err is nil on success.
func (s *SyncStatus) Finished(err error, at time.Time) {
	s.syncing = false
	s.lastErr = err
	if err == nil {
		s.lastOK = at
	}
	s.Refresh()
}

// Refresh redraws the line, so "3m ago" stays current.
func (s *SyncStatus) Refresh() {
	s.view.SetText(s.Render(time.Now()))
}

// Render builds the status text as of now.
func (s *SyncStatus) Render(now time.Time) string {
	label := theme.TagMuted + "Canvas " + theme.TagReset
	switch {
	case !s.configured:
		return label + theme.TagMuted + "○ not set up · c to connect" + theme.TagReset
	case s.syncing:
		return label + theme.TagCyan + "◌ syncing…" + theme.TagReset + s.since(now)
	case s.lastErr != nil && IsOffline(s.lastErr):
		return label + theme.TagYellow + "● offline" + theme.TagReset + s.since(now)
	case s.lastErr != nil:
		return label + theme.TagRed + "● sync failed" + theme.TagReset + theme.TagMuted + " · c for details" + theme.TagReset
	case s.lastOK.IsZero():
		return label + theme.TagMuted + "◌ waiting for first sync" + theme.TagReset
	default:
		return label + theme.TagGreen + "● synced " + dates.Relative(s.lastOK, now) + theme.TagReset
	}
}

// since adds "· last synced 3m ago" when there has been a good sync.
func (s *SyncStatus) since(now time.Time) string {
	if s.lastOK.IsZero() {
		return ""
	}
	return theme.TagMuted + " · last synced " + dates.Relative(s.lastOK, now) + theme.TagReset
}

// IsOffline reports whether err looks like a network problem (no
// connection, DNS failure, timeout) rather than a Canvas error.
func IsOffline(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var dnsErr *net.DNSError
	var opErr *net.OpError
	return errors.As(err, &dnsErr) || errors.As(err, &opErr)
}
