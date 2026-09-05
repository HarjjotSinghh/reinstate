package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

// SignInEmail is one "hopd: email to ..." block from the log email sender
// (HOPD_EMAIL_SENDER=log), the shape internal/email.LogSender.Send writes:
//
//	hopd: email to first-push-acceptance-1@example.com — Sign in to Reinstate Hop
//	Open this link to sign in the device "acceptance-laptop" (darwin-arm64):
//
//	http://127.0.0.1:8082/login/email/oezM6epQ...
//
//	The link expires at ...
type SignInEmail struct {
	To     string
	Device string
	Link   string
}

var (
	emailHeaderRe = regexp.MustCompile(`(?m)^hopd: email to (\S+) — .*$`)
	deviceLineRe  = regexp.MustCompile(`sign in the device "([^"]*)"`)
	magicLinkRe   = regexp.MustCompile(`https?://\S+/login/email/[A-Za-z0-9_\-]+`)
)

// ParseSignInEmails extracts every sign-in email block from raw hopd log
// text (stdout+stderr combined, in the order they were written). It is
// pure text parsing -- no network, no process -- so it is exercised in
// tests against a captured log fixture rather than a live hopd.
func ParseSignInEmails(logText string) []SignInEmail {
	headers := emailHeaderRe.FindAllStringSubmatchIndex(logText, -1)
	out := make([]SignInEmail, 0, len(headers))
	for i, h := range headers {
		blockStart := h[0]
		blockEnd := len(logText)
		if i+1 < len(headers) {
			blockEnd = headers[i+1][0]
		}
		block := logText[blockStart:blockEnd]
		to := logText[h[2]:h[3]]
		e := SignInEmail{To: to}
		if m := deviceLineRe.FindStringSubmatch(block); m != nil {
			e.Device = m[1]
		}
		if m := magicLinkRe.FindString(block); m != "" {
			e.Link = m
		}
		out = append(out, e)
	}
	return out
}

// Approver drives the browser half of email sign-in against a real hopd:
// GET the confirm page (the same request a person's click makes), then, to
// approve, POST the same URL with no body -- confirmPage's form carries no
// field beyond the link token in its action path. A --refuse run performs
// only the GET, so the link is seen and left pending; nothing in hopd's
// current sign-in API lets a caller mark a link explicitly declined (the
// confirm page from a real browser has only one button), so the refused
// path this exercises is the ordinary one: an unapproved link expires on
// its own (HOPD_LOGIN_SESSION_TTL) and the CLI's WaitForApproval sees a
// RefusedError with code link_expired. Shorten HOPD_LOGIN_SESSION_TTL for a
// lab run that wants that quickly.
type Approver struct {
	Client *http.Client
}

func (a *Approver) client() *http.Client {
	if a.Client != nil {
		return a.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// ApprovalResult names what Approve did, for logging.
type ApprovalResult string

const (
	ResultApproved ApprovalResult = "approved"
	ResultDeclined ApprovalResult = "declined (link seen, approval withheld; it will expire on its own)"
)

// Approve fetches link's confirm page and, unless refuse, POSTs it to
// approve the sign-in.
func (a *Approver) Approve(ctx context.Context, link string, refuse bool) (ApprovalResult, error) {
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}
	getResp, err := a.client().Do(getReq)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", link, err)
	}
	getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: unexpected status %s", link, getResp.Status)
	}
	if refuse {
		return ResultDeclined, nil
	}
	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, link, nil)
	if err != nil {
		return "", err
	}
	postResp, err := a.client().Do(postReq)
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", link, err)
	}
	postResp.Body.Close()
	if postResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("POST %s: unexpected status %s", link, postResp.Status)
	}
	return ResultApproved, nil
}
