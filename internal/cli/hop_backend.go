package cli

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/spf13/cobra"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3"
	"github.com/HarjjotSinghh/reinstate/internal/credentials"
	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// hopSeamsContextKey carries the sign-in seams (device-token store and
// friends) to every command that reaches the control plane.
type hopSeamsContextKey struct{}

// hostedHolder remembers the hosted credential source a command opened so
// later error reporting can name the control plane's refusal.
type hostedHolder struct {
	mu     sync.Mutex
	source *hop.Source
}

type hostedHolderContextKey struct{}

func hopSeamsFrom(cmd *cobra.Command) hopCommandOptions {
	if ctx := cmd.Context(); ctx != nil {
		if o, ok := ctx.Value(hopSeamsContextKey{}).(hopCommandOptions); ok {
			return o
		}
	}
	return hopCommandOptions{}
}

func rememberHosted(cmd *cobra.Command, source *hop.Source) {
	if ctx := cmd.Context(); ctx != nil {
		if h, ok := ctx.Value(hostedHolderContextKey{}).(*hostedHolder); ok {
			h.mu.Lock()
			h.source = source
			h.mu.Unlock()
		}
	}
}

// hostedFrom returns the hosted source opened by engineFromConfig, or nil
// for BYO storage.
func hostedFrom(cmd *cobra.Command) *hop.Source {
	if ctx := cmd.Context(); ctx != nil {
		if h, ok := ctx.Value(hostedHolderContextKey{}).(*hostedHolder); ok {
			h.mu.Lock()
			defer h.mu.Unlock()
			return h.source
		}
	}
	return nil
}

// errNotSignedIn is the exit error for a hosted command on a device that
// never ran `rein login`.
func errNotSignedIn() error {
	return NewExitError(ExitAuthStorage, "this device is not signed in to Reinstate Hop; run rein login first")
}

// hostedSession resolves the device token and a control-plane client bound
// to the control plane that issued it.
func hostedSession(cmd *cobra.Command) (credentials.DeviceToken, *hop.Client, error) {
	tok, err := hopSeamsFrom(cmd).tokenStore().GetDeviceToken()
	if errors.Is(err, credentials.ErrNoDeviceToken) {
		return credentials.DeviceToken{}, nil, errNotSignedIn()
	}
	if err != nil {
		return credentials.DeviceToken{}, nil, NewExitError(ExitAuthStorage, err.Error())
	}
	return tok, hop.New(tok.ControlPlaneURL), nil
}

// hostedBackend opens the account's locker: it provisions the locker on
// first use and builds an S3 client whose credentials are minted hourly by
// the control plane. The client then speaks to the locker's endpoint
// directly; no object ever passes through the control plane.
func hostedBackend(cmd *cobra.Command) (*s3.Client, *hop.Source, error) {
	tok, client, err := hostedSession(cmd)
	if err != nil {
		return nil, nil, err
	}
	source := hop.NewSource(client, tok.Token)
	// Remembered before the first call rather than after the last: a
	// command that has to say why it could not open the locker needs the
	// control plane's own error, and that is only reachable through the
	// source it was recorded on.
	rememberHosted(cmd, source)
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	locker, err := source.Locker(ctx)
	if err != nil {
		return nil, nil, hopExitError(err)
	}
	s3client, err := s3.New(context.Background(), s3.Config{
		Endpoint: locker.Endpoint, Region: locker.Region,
		Bucket: locker.Bucket, Prefix: locker.Prefix,
		Credentials: source,
	})
	if err != nil {
		return nil, nil, NewExitError(ExitAuthStorage, err.Error())
	}
	return s3client, source, nil
}

// hopExitError maps a control-plane error to an exit error with a message
// a person can act on.
func hopExitError(err error) error {
	var qe *hop.QuotaError
	switch {
	case errors.As(err, &qe):
		return NewExitError(ExitAuthStorage, fmt.Sprintf("%s; rein hop status shows usage and limits", qe.Error()))
	case errors.Is(err, hop.ErrUnauthorized):
		return NewExitError(ExitAuthStorage, "this device's token was rejected by the control plane (revoked or stale); run rein login again")
	case errors.Is(err, hop.ErrNoLocker):
		return NewExitError(ExitAuthStorage, err.Error())
	case errors.Is(err, hop.ErrStorageUnavailable):
		return NewExitError(ExitRuntime, err.Error())
	case errors.Is(err, hop.ErrPairingExpired), errors.Is(err, hop.ErrPairingDecided), errors.Is(err, hop.ErrPairingConsumed),
		errors.Is(err, hop.ErrPairingRate), errors.Is(err, hop.ErrWrongAccount):
		return NewExitError(ExitAuthStorage, err.Error())
	case errors.Is(err, hop.ErrSelfRevoke), errors.Is(err, hop.ErrDeviceUnknown):
		return NewExitError(ExitUsage, err.Error())
	case errors.Is(err, credentials.ErrNoDeviceToken):
		return errNotSignedIn()
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return err
	}
	return NewExitError(ExitRuntime, err.Error())
}

// hostedError replaces a storage-layer error with the control plane's
// refusal when the hosted source recorded one (the S3 SDK wraps a failed
// credential fetch in several layers of its own prose), and passes any
// other error through unchanged.
func hostedError(source *hop.Source, err error) error {
	if source == nil || err == nil {
		return err
	}
	if last := source.LastError(); last != nil {
		return hopExitError(last)
	}
	return err
}
