package hop

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Location hints the control plane accepts (Cloudflare R2 values). The
// client picks one from the machine's region; apac is the default and what
// India resolves to.
const (
	LocationAPAC    = "apac"
	LocationEastEU  = "eeur"
	LocationWestEU  = "weur"
	LocationEastNA  = "enam"
	LocationWestNA  = "wnam"
	LocationOceania = "oc"
)

// LocationEnv overrides the detected location hint for one process.
const LocationEnv = "REINSTATE_HOP_LOCATION"

// LocationHint picks the storage location nearest to this machine. It
// reads no network: the override variable first, then the local time
// zone's region, defaulting to apac.
func LocationHint() string {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv(LocationEnv))); v != "" {
		return v
	}
	name := time.Local.String()
	if tz := strings.TrimSpace(os.Getenv("TZ")); tz != "" {
		name = tz
	}
	return LocationHintForZone(name)
}

// LocationHintForZone maps an IANA time-zone name to a location hint.
func LocationHintForZone(zone string) string {
	region, city, _ := strings.Cut(zone, "/")
	switch region {
	case "Europe", "Atlantic", "Africa":
		switch city {
		case "Moscow", "Istanbul", "Kyiv", "Kiev", "Minsk", "Warsaw", "Helsinki", "Bucharest", "Sofia", "Athens", "Riga", "Tallinn", "Vilnius", "Samara", "Volgograd", "Simferopol", "Chisinau":
			return LocationEastEU
		}
		return LocationWestEU
	case "America", "Canada", "US", "Mexico", "Cuba", "Jamaica":
		switch city {
		case "Los_Angeles", "Vancouver", "Tijuana", "Phoenix", "Denver", "Edmonton", "Boise", "Juneau", "Anchorage", "Adak", "Whitehorse", "Dawson", "Yellowknife", "Pacific", "Mountain", "Alaska", "Hawaii", "Arizona", "Hermosillo", "Mazatlan", "Chihuahua":
			return LocationWestNA
		}
		return LocationEastNA
	case "Pacific", "Australia", "Antarctica":
		if city == "Honolulu" {
			return LocationWestNA
		}
		return LocationOceania
	case "Asia", "Indian":
		return LocationAPAC
	}
	return LocationAPAC
}

// Locker describes the account's bucket as the control plane reports it.
type Locker struct {
	Endpoint     string `json:"endpoint"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	Prefix       string `json:"prefix"`
	LocationHint string `json:"location_hint"`
	Plan         string `json:"plan"`
	CreatedAt    string `json:"created_at"`
	FirstPushAt  string `json:"first_push_at,omitempty"`
	Devices      int    `json:"devices"`
	Usage        Usage  `json:"usage"`
	Quota        Quota  `json:"quota"`
}

// Usage is the locker's measured size.
type Usage struct {
	Bytes      int64  `json:"bytes"`
	Objects    int64  `json:"objects"`
	ObservedAt string `json:"observed_at,omitempty"`
}

// Quota is the plan's limits; zero means unlimited.
type Quota struct {
	StorageBytes int64 `json:"storage_bytes"`
	Devices      int   `json:"devices"`
	MintsPerHour int   `json:"mints_per_hour"`
}

// LockerCredentials are one minted credential set bound to the locker.
type LockerCredentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
	ExpiresAt       string `json:"expires_at"`
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
}

// Expires parses ExpiresAt; zero when absent or malformed.
func (c LockerCredentials) Expires() time.Time {
	t, err := time.Parse(time.RFC3339, c.ExpiresAt)
	if err != nil {
		return time.Time{}
	}
	return t
}

// Quota kinds carried by QuotaError.
const (
	QuotaStorage  = "storage"
	QuotaDevices  = "devices"
	QuotaPushRate = "push-rate"
)

// QuotaError is a refusal because the account is over one quota.
type QuotaError struct {
	Kind    string
	Message string
}

func (e *QuotaError) Error() string {
	return fmt.Sprintf("locker over quota (%s): %s", e.Kind, e.Message)
}

// ErrNoLocker reports that the account has no locker yet.
var ErrNoLocker = errors.New("no locker has been provisioned for this account yet; the first push creates it")

// ErrStorageUnavailable reports that the control plane could not reach the
// storage provider.
var ErrStorageUnavailable = errors.New("the control plane could not reach the storage provider; try again shortly")

// ProvisionLocker creates the account's locker if needed and returns it.
func (c *Client) ProvisionLocker(ctx context.Context, token string) (Locker, error) {
	var out Locker
	if err := c.do(ctx, http.MethodPost, "/v1/locker", token, nil, &out); err != nil {
		return Locker{}, lockerError(err)
	}
	if out.Endpoint == "" || out.Bucket == "" {
		return Locker{}, errors.New("control plane returned an incomplete locker")
	}
	return out, nil
}

// LockerStatus returns the locker without provisioning it.
func (c *Client) LockerStatus(ctx context.Context, token string) (Locker, error) {
	var out Locker
	if err := c.do(ctx, http.MethodGet, "/v1/locker", token, nil, &out); err != nil {
		return Locker{}, lockerError(err)
	}
	return out, nil
}

// MintCredentials asks for a fresh credential set bound to the locker.
func (c *Client) MintCredentials(ctx context.Context, token string) (LockerCredentials, error) {
	var out LockerCredentials
	if err := c.do(ctx, http.MethodPost, "/v1/locker/credentials", token, nil, &out); err != nil {
		return LockerCredentials{}, lockerError(err)
	}
	if out.AccessKeyID == "" || out.SecretAccessKey == "" {
		return LockerCredentials{}, errors.New("control plane returned incomplete credentials")
	}
	return out, nil
}

// ReportFirstPush tells the control plane a push completed. It reports
// whether this was the account's first.
func (c *Client) ReportFirstPush(ctx context.Context, token string) (bool, error) {
	var out struct {
		First bool `json:"first"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/locker/first-push", token, nil, &out); err != nil {
		return false, lockerError(err)
	}
	return out.First, nil
}

// lockerError turns the control plane's typed refusals into typed errors.
func lockerError(err error) error {
	var he *Error
	if !errors.As(err, &he) {
		return err
	}
	switch he.Code {
	case CodeQuotaStorage:
		return &QuotaError{Kind: QuotaStorage, Message: he.Message}
	case CodeQuotaDevices:
		// The same code a refused sign-in carries (refusal.go): one plan
		// limit, one word for it, whichever route ran into it.
		return &QuotaError{Kind: QuotaDevices, Message: he.Message}
	case CodeQuotaPushRate:
		return &QuotaError{Kind: QuotaPushRate, Message: he.Message}
	case "no_locker":
		return ErrNoLocker
	case "storage_unavailable":
		return ErrStorageUnavailable
	}
	if he.Status == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	return err
}
