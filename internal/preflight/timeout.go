// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package preflight

import (
	"context"
	"time"
)

// WithTimeout returns a Verifier that runs the same checks as v but bound to
// timeout instead of whatever budget v itself was configured with.
//
// It exists for background callers — the switcher's readiness prober,
// chiefly — that must not compete for the interactive launch path's strict
// wall clock (DefaultVerifierTimeout). The launch path keeps calling the
// verifier it was given directly; only a caller that explicitly wants a
// different budget wraps it with this.
//
// Service is the production Verifier, and overriding Options.Timeout is what
// actually lengthens or shortens Verify's *overall* deadline — a derived
// context alone cannot: context.WithTimeout takes the *earlier* of a parent
// deadline and its own, so a longer parent deadline can never stretch a
// shorter child one. That overall deadline is not the whole story, though:
// Verify's own workspace, agent-version, and runtime observers each carry
// their own Options.Timeout (Options.Workspace.Timeout,
// Options.Agent.Timeout, Options.Runtime.Timeout), and remainingTimeout
// (verify.go) falls back to the package-level DefaultVerifierTimeout
// constant — not to whatever the *overall* Options.Timeout was set to —
// whenever one of those is left zero, which is how every production Service
// leaves them. Setting only the top-level Options.Timeout would raise the
// ceiling for the call as a whole while leaving every individual observer
// still racing the launch path's own two-second constant underneath it, so
// this sets all four fields to the same timeout: the whole call, and every
// observer inside it, now shares one budget instead of two.
//
// A Verifier that manages its own budget some other way (a test fake,
// chiefly) instead gets a context bounded to timeout, which is a no-op for a
// fake that answers immediately and does not weaken a real one that would
// already be racing the same wall clock through ctx.
func WithTimeout(v Verifier, timeout time.Duration) Verifier {
	switch typed := v.(type) {
	case nil:
		return nil
	case Service:
		return withTimeoutOptions(typed, timeout)
	case *Service:
		if typed == nil {
			return v
		}
		return withTimeoutOptions(*typed, timeout)
	default:
		return boundedVerifier{Verifier: v, timeout: timeout}
	}
}

func withTimeoutOptions(service Service, timeout time.Duration) Service {
	service.Options.Timeout = timeout
	service.Options.Workspace.Timeout = timeout
	service.Options.Agent.Timeout = timeout
	service.Options.Runtime.Timeout = timeout
	return service
}

// boundedVerifier bounds an arbitrary Verifier's context rather than its
// internal Options, for the Verifier implementations (test fakes, mainly)
// that are not a preflight.Service and so have no Options.Timeout to set.
type boundedVerifier struct {
	Verifier
	timeout time.Duration
}

func (b boundedVerifier) Verify(ctx context.Context, input Input) (Report, error) {
	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	return b.Verifier.Verify(ctx, input)
}
