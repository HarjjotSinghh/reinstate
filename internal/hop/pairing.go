package hop

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Pairing protocol versions. Version 0 is not emitted; on received requests
// it means version 1 for compatibility with the original unversioned wire.
const (
	PairingVersion1 = 1
	PairingVersion2 = 2
)

// Pairing-request statuses.
const (
	PairingPending  = "pending"
	PairingApproved = "approved"
	PairingConsumed = "consumed"
	PairingExpired  = "expired"
)

// Pairing-request refusal codes.
const (
	CodePairingExpired = "pairing_expired"
	CodePairingDecided = "pairing_decided"
	CodePairingRate    = "pairing_rate"
	CodeWrongAccount   = "wrong_account"
	CodeSelfRevoke     = "self_revoke"
	CodeDeviceUnknown  = "device_unknown"
)

// PairingRequest is one device-approval request as the control plane
// relays it. PublicKey, Salt, and Binding are published by the joining
// device; the control plane cannot check or forge them.
type PairingRequest struct {
	ID              string `json:"id"`
	Version         int    `json:"version"`
	Status          string `json:"status"`
	Device          Device `json:"device"`
	PublicKey       string `json:"public_key"`
	Salt            string `json:"salt"`
	Binding         string `json:"binding"`
	CreatedAt       string `json:"created_at"`
	ExpiresAt       string `json:"expires_at"`
	IntervalSeconds int    `json:"interval_seconds,omitempty"`
}

// ProtocolVersion returns the request's effective pairing protocol. The
// original wire omitted version, so both a missing field and explicit zero
// mean v1. Values other than 1 and 2 are never guessed.
func (p PairingRequest) ProtocolVersion() (int, error) {
	return protocolVersion(p.Version)
}

func protocolVersion(version int) (int, error) {
	if version == 0 {
		return PairingVersion1, nil
	}
	if version != PairingVersion1 && version != PairingVersion2 {
		return 0, fmt.Errorf("control plane returned unsupported pairing version %d (want 1 or 2)", version)
	}
	return version, nil
}

// PairingPayload is what the joining device collects exactly once.
type PairingPayload struct {
	Version       int
	Payload       string
	KeyGeneration int
	ApprovedBy    Device
}

// Pairing errors.
var (
	ErrPairingExpired  = errors.New("the pairing request expired before it was approved and collected; run rein account join again on the new device")
	ErrPairingDecided  = errors.New("the pairing request was already approved, collected, or cancelled")
	ErrPairingConsumed = errors.New("the pairing payload was already collected")
	ErrPairingRate     = errors.New("the pairing request was polled too often; run rein account join again")
	ErrWrongAccount    = errors.New("the request belongs to another account")
	ErrSelfRevoke      = errors.New("a device cannot revoke itself; revoke it from another enrolled device")
	ErrDeviceUnknown   = errors.New("no such device on this account")
)

// Revocation is the control plane's answer to a revocation: the device as
// it now stands, and whether this call did the revoking (false when the
// device was already revoked, which is not an error).
type Revocation struct {
	Device  Device `json:"device"`
	Revoked bool   `json:"revoked"`
}

const (
	RevocationRequestPending   = "pending"
	RevocationRequestCancelled = "cancelled"
	RevocationRequestConfirmed = "confirmed"

	CodeGenerationNotNewer = "generation_not_newer"
	CodeRevocationDecided  = "revocation_decided"
)

var (
	ErrGenerationNotNewer = errors.New("the reported key generation is not newer than the pending revocation")
	ErrRevocationDecided  = errors.New("the device revocation request is no longer pending")
)

type DeviceRevocationRequest struct {
	ID                  string `json:"id"`
	Status              string `json:"status"`
	Target              Device `json:"target"`
	RequestedGeneration int    `json:"requested_generation"`
	RequestedAt         string `json:"requested_at"`
	ConfirmedAt         string `json:"confirmed_at,omitempty"`
	ConfirmedBy         string `json:"confirmed_by,omitempty"`
	ConfirmedGeneration int    `json:"confirmed_generation,omitempty"`
}

// RevokeDevice tells the control plane that device id is revoked: its token
// is refused from then on and credential minting no longer counts it. The
// call is idempotent. It carries no key material; the key generation
// rollover happens in the keyring before this is called.
func (c *Client) RevokeDevice(ctx context.Context, token, id string) (Revocation, error) {
	var out Revocation
	if err := c.do(ctx, http.MethodDelete, "/v1/devices/"+url.PathEscape(id), token, nil, &out); err != nil {
		return Revocation{}, pairingError(err)
	}
	if out.Device.ID == "" {
		return Revocation{}, errors.New("control plane answered the revocation without the device")
	}
	return out, nil
}

func (c *Client) PendingDeviceRevocations(ctx context.Context, token string) ([]DeviceRevocationRequest, error) {
	var out struct {
		Requests []DeviceRevocationRequest `json:"requests"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/device-revocation-requests", token, nil, &out); err != nil {
		var he *Error
		if errors.As(err, &he) && he.Status == http.StatusNotFound {
			return []DeviceRevocationRequest{}, nil
		}
		return nil, pairingError(err)
	}
	return out.Requests, nil
}

func (c *Client) ConfirmDeviceRevocation(ctx context.Context, token, id string, generation int) (DeviceRevocationRequest, error) {
	var out DeviceRevocationRequest
	if err := c.do(ctx, http.MethodPost, "/v1/device-revocation-requests/"+url.PathEscape(id)+"/confirm", token, map[string]int{"generation": generation}, &out); err != nil {
		var he *Error
		if errors.As(err, &he) {
			switch he.Code {
			case CodeGenerationNotNewer:
				return DeviceRevocationRequest{}, ErrGenerationNotNewer
			case CodeRevocationDecided:
				return DeviceRevocationRequest{}, ErrRevocationDecided
			}
		}
		return DeviceRevocationRequest{}, pairingError(err)
	}
	if out.ID == "" || out.Target.ID == "" || out.Status != RevocationRequestConfirmed {
		return DeviceRevocationRequest{}, errors.New("control plane answered the revocation confirmation incompletely")
	}
	return out, nil
}

// CreatePairing opens a v2 pairing request for this device. Version is a JSON
// integer on the wire; an old control plane that drops it cannot safely relay
// the v2 binding to an approver and is refused explicitly.
func (c *Client) CreatePairing(ctx context.Context, token, publicKey, salt, binding string) (PairingRequest, error) {
	var out PairingRequest
	err := c.do(ctx, http.MethodPost, "/v1/pairing", token, map[string]any{"version": PairingVersion2, "public_key": publicKey, "salt": salt, "binding": binding}, &out)
	if err != nil {
		return PairingRequest{}, pairingError(err)
	}
	version, versionErr := out.ProtocolVersion()
	if out.ID == "" || versionErr != nil || version != PairingVersion2 {
		detail := ""
		if versionErr != nil {
			detail = ": " + versionErr.Error()
		} else if version != PairingVersion2 {
			detail = fmt.Sprintf(": returned pairing version %d for a v2 request", version)
		}
		return PairingRequest{}, errors.New("control plane returned an incomplete pairing request" + detail)
	}
	return out, nil
}

// PendingPairings lists the account's open requests.
func (c *Client) PendingPairings(ctx context.Context, token string) ([]PairingRequest, error) {
	var out struct {
		Requests []PairingRequest `json:"requests"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/pairing", token, nil, &out); err != nil {
		return nil, pairingError(err)
	}
	return out.Requests, nil
}

// GetPairing fetches one request.
func (c *Client) GetPairing(ctx context.Context, token, id string) (PairingRequest, error) {
	var out PairingRequest
	if err := c.do(ctx, http.MethodGet, "/v1/pairing/"+id, token, nil, &out); err != nil {
		return PairingRequest{}, pairingError(err)
	}
	return out, nil
}

// ApprovePairing relays the sealed root key for request id and requires the
// relay to confirm the same cryptographic protocol the approver used.
func (c *Client) ApprovePairing(ctx context.Context, token, id, payload string, generation, expectedVersion int) error {
	var out struct {
		Status  string `json:"status"`
		Version int    `json:"version"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/pairing/"+id+"/approve", token, map[string]any{"payload": payload, "key_generation": generation}, &out); err != nil {
		return pairingError(err)
	}
	version, err := protocolVersion(out.Version)
	if err != nil {
		return err
	}
	if version != expectedVersion {
		return fmt.Errorf("control plane approved pairing %s as version %d, want version %d", id, version, expectedVersion)
	}
	return nil
}

// ExpirePairing cancels a pending request.
func (c *Client) ExpirePairing(ctx context.Context, token, id string) error {
	return pairingError(c.do(ctx, http.MethodPost, "/v1/pairing/"+id+"/expire", token, nil, nil))
}

type claimResponse struct {
	Status        string  `json:"status"`
	Version       int     `json:"version"`
	Payload       string  `json:"payload,omitempty"`
	KeyGeneration int     `json:"key_generation,omitempty"`
	ApprovedBy    *Device `json:"approved_by,omitempty"`
}

// ClaimPairing polls request id once. The payload is non-nil exactly once.
func (c *Client) ClaimPairing(ctx context.Context, token, id string, expectedVersion int) (string, *PairingPayload, error) {
	var out claimResponse
	err := c.do(ctx, http.MethodPost, "/v1/pairing/"+id+"/claim", token, nil, &out)
	var he *Error
	gone := errors.As(err, &he) && he.Status == http.StatusGone && out.Status != ""
	if err != nil && !gone {
		return "", nil, pairingError(err)
	}
	version, versionErr := protocolVersion(out.Version)
	if versionErr != nil {
		return "", nil, versionErr
	}
	if version != expectedVersion {
		return "", nil, fmt.Errorf("control plane answered pairing %s as version %d, want version %d", id, version, expectedVersion)
	}
	if gone {
		return out.Status, nil, nil
	}
	if out.Status == PairingApproved {
		if out.Payload == "" || out.KeyGeneration <= 0 {
			return "", nil, errors.New("control plane approved the pairing without a payload")
		}
		p := &PairingPayload{Version: version, Payload: out.Payload, KeyGeneration: out.KeyGeneration}
		if out.ApprovedBy != nil {
			p.ApprovedBy = *out.ApprovedBy
		}
		return out.Status, p, nil
	}
	return out.Status, nil, nil
}

// WaitForPairing polls until the request is approved, expired, or ctx ends.
func (c *Client) WaitForPairing(ctx context.Context, token string, req PairingRequest, sleep func(context.Context, time.Duration) error) (*PairingPayload, error) {
	version, err := req.ProtocolVersion()
	if err != nil {
		return nil, err
	}
	interval := time.Duration(req.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if sleep == nil {
		sleep = defaultSleep
	}
	for {
		status, payload, err := c.ClaimPairing(ctx, token, req.ID, version)
		if err != nil {
			return nil, err
		}
		switch status {
		case PairingApproved:
			return payload, nil
		case PairingPending:
		case PairingExpired:
			return nil, ErrPairingExpired
		case PairingConsumed:
			return nil, ErrPairingConsumed
		default:
			return nil, fmt.Errorf("pairing request is %s; run rein account join again", status)
		}
		if err := sleep(ctx, interval); err != nil {
			return nil, err
		}
	}
}

// Devices lists the account's enrolled devices.
func (c *Client) Devices(ctx context.Context, token string) ([]Device, error) {
	var out struct {
		Devices []Device `json:"devices"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/devices", token, nil, &out); err != nil {
		return nil, pairingError(err)
	}
	return out.Devices, nil
}

func pairingError(err error) error {
	var he *Error
	if !errors.As(err, &he) {
		return err
	}
	switch he.Code {
	case CodePairingExpired:
		return ErrPairingExpired
	case CodePairingDecided:
		return ErrPairingDecided
	case CodePairingRate:
		return ErrPairingRate
	case CodeWrongAccount:
		return ErrWrongAccount
	case CodeSelfRevoke:
		return ErrSelfRevoke
	case CodeDeviceUnknown:
		return ErrDeviceUnknown
	}
	if he.Status == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	return err
}
