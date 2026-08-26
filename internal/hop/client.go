// Package hop is the client of the Reinstate Hop control plane: the service
// that signs accounts in and enrols devices. The protocol is public (see
// docs/hop.md); the control plane never receives a key or plaintext.
package hop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// DefaultURL is the production control plane.
const DefaultURL = "https://hop.reinstate.dev"

// URLEnv overrides the control-plane URL for one process.
const URLEnv = "REINSTATE_HOP_URL"

// Sign-in methods.
const (
	MethodGitHub = "github"
	MethodEmail  = "email"
)

// Login-session statuses.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusExpired  = "expired"
	StatusConsumed = "consumed"
)

// ResolveURL picks the control-plane URL: the environment first, then the
// optional config value, then the production default.
func ResolveURL(cfg *schema.Config) string {
	if v := strings.TrimSpace(os.Getenv(URLEnv)); v != "" {
		return strings.TrimRight(v, "/")
	}
	if cfg != nil && strings.TrimSpace(cfg.Hop.URL) != "" {
		return strings.TrimRight(strings.TrimSpace(cfg.Hop.URL), "/")
	}
	return DefaultURL
}

// Client talks to one control plane.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a client for baseURL with a bounded HTTP timeout.
func New(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// DeviceInfo describes this machine to the control plane.
type DeviceInfo struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	// LocationHint is where this device would like the account's locker to
	// live (see LocationHint). It matters only for the sign-in that creates
	// the account.
	LocationHint string `json:"location_hint,omitempty"`
}

// Account is the sign-in identity as shown to its owner.
type Account struct {
	ID           string `json:"id"`
	Email        string `json:"email,omitempty"`
	GitHubLogin  string `json:"github_login,omitempty"`
	Plan         string `json:"plan,omitempty"`
	LocationHint string `json:"location_hint,omitempty"`
	CreatedAt    string `json:"created_at"`
}

// Device is one enrolled machine.
type Device struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Platform     string `json:"platform"`
	LocationHint string `json:"location_hint,omitempty"`
	CreatedAt    string `json:"created_at"`
	LastSeenAt   string `json:"last_seen_at"`
}

// LoginSession is a sign-in attempt awaiting browser approval.
type LoginSession struct {
	ID              string `json:"session_id"`
	PollSecret      string `json:"poll_secret"`
	Method          string `json:"method"`
	VerificationURL string `json:"verification_url,omitempty"`
	ExpiresAt       string `json:"expires_at"`
	IntervalSeconds int    `json:"interval_seconds"`
}

// Approval is the single successful poll result.
type Approval struct {
	DeviceToken string
	Account     Account
	Device      Device
}

// Identity answers a device token.
type Identity struct {
	Account Account `json:"account"`
	Device  Device  `json:"device"`
}

// Error is a non-2xx answer from the control plane.
type Error struct {
	Status  int
	Message string
	// Code names the kind of refusal when the control plane sent one.
	Code string
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("control plane answered %d", e.Status)
	}
	return e.Message
}

// ErrUnauthorized reports a token the control plane no longer accepts.
var ErrUnauthorized = errors.New("device token rejected by the control plane")

// Unreachable reports whether err is a failure to reach the control plane
// at all — no DNS answer, no route, no TLS handshake, no reply in time —
// as opposed to an answer it gave. The distinction matters wherever a
// caller has to tell a check that could not run from a check that failed:
// a service nobody could reach says nothing about the property being
// checked, and reporting it as a failure of that property is a false
// alarm.
//
// Every non-2xx answer leaves do as *Error and every transport failure as
// the *url.Error net/http wraps it in, so the two are cleanly separable.
func Unreachable(err error) bool {
	if err == nil {
		return false
	}
	var answered *Error
	if errors.As(err, &answered) {
		return false
	}
	var transport *url.Error
	return errors.As(err, &transport)
}

// StartLogin opens a login session. For email sign-in the link is sent to
// addr and VerificationURL stays empty.
func (c *Client) StartLogin(ctx context.Context, method, addr string, device DeviceInfo) (LoginSession, error) {
	var out LoginSession
	body := map[string]any{"method": method, "device": device}
	if addr != "" {
		body["email"] = addr
	}
	if err := c.do(ctx, http.MethodPost, "/v1/login/sessions", "", body, &out); err != nil {
		return LoginSession{}, err
	}
	if out.ID == "" || out.PollSecret == "" {
		return LoginSession{}, errors.New("control plane returned an incomplete login session")
	}
	return out, nil
}

type pollResponse struct {
	Status      string   `json:"status"`
	DeviceToken string   `json:"device_token,omitempty"`
	Account     *Account `json:"account,omitempty"`
	Device      *Device  `json:"device,omitempty"`
}

// Poll reports the session status. The Approval is non-nil exactly once.
func (c *Client) Poll(ctx context.Context, s LoginSession) (string, *Approval, error) {
	var out pollResponse
	err := c.do(ctx, http.MethodPost, "/v1/login/sessions/"+s.ID+"/poll", "", map[string]string{"poll_secret": s.PollSecret}, &out)
	var he *Error
	if errors.As(err, &he) && he.Status == http.StatusGone && out.Status != "" {
		return out.Status, nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	if out.Status == StatusApproved {
		if out.DeviceToken == "" || out.Account == nil || out.Device == nil {
			return "", nil, errors.New("control plane approved the login without a device token")
		}
		return out.Status, &Approval{DeviceToken: out.DeviceToken, Account: *out.Account, Device: *out.Device}, nil
	}
	return out.Status, nil, nil
}

// WaitForApproval polls until the session is approved, expired, or ctx ends.
// sleep is injectable so tests never wait.
func (c *Client) WaitForApproval(ctx context.Context, s LoginSession, sleep func(context.Context, time.Duration) error) (*Approval, error) {
	interval := time.Duration(s.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if sleep == nil {
		sleep = defaultSleep
	}
	for {
		status, approval, err := c.Poll(ctx, s)
		if err != nil {
			return nil, err
		}
		switch status {
		case StatusApproved:
			return approval, nil
		case StatusPending:
		case StatusExpired:
			return nil, errors.New("the sign-in link expired before it was used; run login again")
		default:
			return nil, fmt.Errorf("login session is %s; run login again", status)
		}
		if err := sleep(ctx, interval); err != nil {
			return nil, err
		}
	}
}

func defaultSleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Whoami resolves a device token to its account and device.
func (c *Client) Whoami(ctx context.Context, token string) (Identity, error) {
	var out Identity
	err := c.do(ctx, http.MethodGet, "/v1/whoami", token, nil, &out)
	var he *Error
	if errors.As(err, &he) && he.Status == http.StatusUnauthorized {
		return Identity{}, ErrUnauthorized
	}
	if err != nil {
		return Identity{}, err
	}
	return out, nil
}

func (c *Client) do(ctx context.Context, method, path, token string, in, out any) error {
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("reach control plane %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if out != nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, out)
	}
	if resp.StatusCode/100 != 2 {
		var e struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		_ = json.Unmarshal(raw, &e)
		return &Error{Status: resp.StatusCode, Message: e.Error, Code: e.Code}
	}
	return nil
}
