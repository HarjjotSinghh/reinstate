package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// approveOptions configures `hoplab approve`.
type approveOptions struct {
	Root    string // reads the log path from LabState here, unless LogPath is set
	LogPath string
	Email   string // only approve links to this address; empty means any
	Count   int    // stop after this many approvals (or refusals)
	Timeout time.Duration
	Refuse  bool
}

// runApprove tails the hopd log (re-read on each poll -- lab-scale logs are
// small, and this avoids fragile file-tail bookkeeping) and approves every
// new sign-in email it finds, up to o.Count, printing each decision to out.
// It returns once o.Count is reached or o.Timeout elapses.
func runApprove(ctx context.Context, o approveOptions, out io.Writer, a *Approver) error {
	logPath := o.LogPath
	if logPath == "" {
		s, err := loadState(o.Root)
		if err != nil {
			return fmt.Errorf("load lab state under %s: %w (pass -log directly if the lab was started elsewhere)", o.Root, err)
		}
		logPath = s.HopdLog
	}
	count := o.Count
	if count <= 0 {
		count = 1
	}
	deadline := time.Now().Add(o.Timeout)
	seen := map[string]bool{}
	approved := 0
	for {
		raw, err := os.ReadFile(logPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read %s: %w", logPath, err)
		}
		for _, email := range ParseSignInEmails(string(raw)) {
			if email.Link == "" || seen[email.Link] {
				continue
			}
			if o.Email != "" && !strings.EqualFold(email.To, o.Email) {
				continue
			}
			seen[email.Link] = true
			result, err := a.Approve(ctx, email.Link, o.Refuse)
			if err != nil {
				_, _ = fmt.Fprintf(out, "hoplab approve: device %q (%s): FAILED: %v\n", email.Device, email.To, err)
				continue
			}
			_, _ = fmt.Fprintf(out, "hoplab approve: device %q (%s): %s\n", email.Device, email.To, result)
			approved++
			if approved >= count {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s having approved %d/%d sign-in(s)", o.Timeout, approved, count)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}
