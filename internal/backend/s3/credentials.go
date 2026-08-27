package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// Credentials are one set of S3-compatible access keys. Expires is zero when
// the keys never expire (the BYO static case).
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	// SessionToken is set for temporary credentials and empty otherwise.
	SessionToken string
	Expires      time.Time
}

// CredentialSource yields credentials on demand. Implementations may return a
// different, fresher set on every call; the backend asks again when the
// previous set expires or is rejected by the storage endpoint.
type CredentialSource interface {
	Credentials(ctx context.Context) (Credentials, error)
}

// StaticSource holds one non-expiring credential set. It is the default for
// BYO storage, wrapping the keys loaded from the OS keyring or environment.
type StaticSource struct {
	creds Credentials
}

// Static returns a CredentialSource for fixed, non-expiring keys.
func Static(accessKeyID, secretAccessKey string) StaticSource {
	return StaticSource{creds: Credentials{AccessKeyID: accessKeyID, SecretAccessKey: secretAccessKey}}
}

// StaticCredentials wraps one already-minted credential set (session token
// included) so a client can be built that never asks for another; rein sync
// verify uses it to probe the reference locker with exactly the credential
// the locker accepted.
func StaticCredentials(c Credentials) StaticSource {
	return StaticSource{creds: c}
}

// Credentials implements CredentialSource.
func (s StaticSource) Credentials(context.Context) (Credentials, error) {
	return s.creds, nil
}

// sourceProvider adapts a CredentialSource to the AWS SDK provider interface.
// The SDK's CredentialsCache wraps it and re-calls Retrieve once Expires has
// passed (minus the expiry window), which gives proactive refresh for free.
type sourceProvider struct {
	source CredentialSource
}

func (p sourceProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	c, err := p.source.Credentials(ctx)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("s3 credentials: %w", err)
	}
	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return aws.Credentials{}, fmt.Errorf("s3 credentials: source returned empty keys")
	}
	out := aws.Credentials{
		AccessKeyID:     c.AccessKeyID,
		SecretAccessKey: c.SecretAccessKey,
		SessionToken:    c.SessionToken,
		Source:          "reinstate",
	}
	if !c.Expires.IsZero() {
		out.CanExpire = true
		out.Expires = c.Expires
	}
	return out, nil
}

// refreshExpiryWindow is how long before Expires the cache treats a credential
// as stale and asks the source again. Hourly locker credentials are refreshed
// well before the storage endpoint starts rejecting them.
const refreshExpiryWindow = time.Minute
