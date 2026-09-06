package hop

import (
	"context"
	"errors"
	"sync"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3"
)

// Source is the hosted-tier CredentialSource: it provisions the account's
// locker on first use and mints hourly credentials bound to it from the
// control plane with the device token. The S3 backend refreshes through it
// when a credential expires or is rejected, so a push that outlives one
// credential set simply asks for the next.
type Source struct {
	client *Client
	token  string

	mu      sync.Mutex
	locker  *Locker
	mints   int
	lastErr error
}

// NewSource returns a Source for one device token.
func NewSource(client *Client, token string) *Source {
	return &Source{client: client, token: token}
}

// Locker returns the account's locker, provisioning it on the first call.
// An existing locker is read with GET /v1/locker; only ErrNoLocker leads to
// POST /v1/locker, so routine commands never hit the provisioning path.
func (s *Source) Locker(ctx context.Context) (Locker, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locker != nil {
		return *s.locker, nil
	}
	l, err := s.client.LockerStatus(ctx, s.token)
	if errors.Is(err, ErrNoLocker) {
		l, err = s.client.ProvisionLocker(ctx, s.token)
	}
	if err != nil {
		s.lastErr = err
		return Locker{}, err
	}
	s.locker = &l
	return l, nil
}

// Credentials implements s3.CredentialSource.
func (s *Source) Credentials(ctx context.Context) (s3.Credentials, error) {
	c, err := s.client.MintCredentials(ctx, s.token)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.lastErr = err
		return s3.Credentials{}, err
	}
	s.mints++
	s.lastErr = nil
	return s3.Credentials{
		AccessKeyID:     c.AccessKeyID,
		SecretAccessKey: c.SecretAccessKey,
		SessionToken:    c.SessionToken,
		Expires:         c.Expires(),
	}, nil
}

// ReportFirstPush reports a completed push when the locker had none
// before; it is a no-op once the control plane has recorded one.
func (s *Source) ReportFirstPush(ctx context.Context) error {
	s.mu.Lock()
	l := s.locker
	s.mu.Unlock()
	if l != nil && l.FirstPushAt != "" {
		return nil
	}
	_, err := s.client.ReportFirstPush(ctx, s.token)
	if err == nil {
		s.mu.Lock()
		if s.locker != nil && s.locker.FirstPushAt == "" {
			s.locker.FirstPushAt = "reported"
		}
		s.mu.Unlock()
	}
	return err
}

// Mints reports how many credential sets were minted through this source.
func (s *Source) Mints() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mints
}

// LastError returns the control plane's most recent refusal, so a caller
// that only sees the storage layer's wrapped error can report the cause.
func (s *Source) LastError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastErr
}

// IsRefusal reports whether err is a control-plane refusal a person must
// act on (sign in again, free space, wait) rather than a transient fault.
func IsRefusal(err error) bool {
	var qe *QuotaError
	return errors.As(err, &qe) || errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrNoLocker)
}

var _ s3.CredentialSource = (*Source)(nil)
