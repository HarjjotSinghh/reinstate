package hop

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// The account key-generation floor.
//
// The keyring's own defences are per device: a device refuses a keyring
// below the generation *it* has already unwrapped. That leaves the device
// which has not yet read the rollover, because it has no local knowledge a
// rollback would contradict. The control plane is the party that can see
// the rollover such a device missed — it authenticates every device and
// refuses a revoked one — so it carries the account's key generation as a
// number that only ever goes up, and every device checks the keyring
// against it before acting on it.
//
// What this buys, precisely:
//
//   - Against the revoked device — the adversary revocation exists to stop,
//     and the realistic one, a stolen laptop — it closes the gap. The
//     control plane refuses that device's token, so that device has no way
//     to read or raise the floor, and every other device is told the
//     account has moved on whether or not it has read the keyring.
//   - Against an operator holding **both** the control plane and the bucket
//     it adds nothing: that party can serve whatever floor it likes, to a
//     device that has never confirmed one. The recovery-code signature on
//     every generation and the local anchor in account.json are what cover
//     that adversary, as far as they cover it.
//   - A device that *has* confirmed a floor keeps the higher of that and
//     whatever it is told next (see the client's caller), so an answer that
//     goes backwards is not accepted from anyone.
//   - The control plane holds no key and sees no keyring, so a reported
//     rollover is a claim rather than an observation. It orders generations;
//     authenticating one is the keyring's job, and stays here.
//
// See docs/hop.md and docs/security-model.md, which state all of that.

// KeyGenerationPath is the control-plane route that serves and raises the
// account's key generation floor.
const KeyGenerationPath = "/v1/account/key-generation"

// CodeKeyGenerationRange is the refusal code for a reported generation
// outside the range the control plane will store.
const CodeKeyGenerationRange = "key_generation_range"

// KeyGeneration is the account's key generation floor as the control plane
// holds it. Generation is 0 on an account no revocation has touched, which
// every keyring satisfies.
type KeyGeneration struct {
	Generation int `json:"generation"`
	// RaisedAt and RaisedBy are absent while the floor is 0.
	RaisedAt string `json:"raised_at,omitempty"`
	RaisedBy string `json:"raised_by,omitempty"`
	// Raised is the POST answer's report of whether that call moved the
	// floor. A report at or below the floor is not an error: it answers
	// with the floor as it stands and Raised false, which is how a device
	// that has not seen another device's rollover learns it is behind.
	Raised bool `json:"raised,omitempty"`
}

// ErrNoKeyGenerationFloor reports a control plane that does not serve the
// floor at all: a deployment older than the route, answering 404. It is not
// a refusal of this device, and callers fall back to the last floor this
// device had confirmed rather than treating the account as being at
// generation 0.
var ErrNoKeyGenerationFloor = errors.New("this control plane does not carry the account's key generation floor")

// ErrKeyGenerationRange reports a generation the control plane will not
// store. It bounds what one enrolled device can do to its own account's
// floor; it is not a check that a rollover happened, which the control
// plane cannot see.
var ErrKeyGenerationRange = errors.New("the control plane refused that key generation as out of range")

// KeyGenerationFloor reads the account's key generation floor. A revoked or
// unknown token gets ErrUnauthorized; a control plane without the route
// gets ErrNoKeyGenerationFloor.
func (c *Client) KeyGenerationFloor(ctx context.Context, token string) (KeyGeneration, error) {
	var out KeyGeneration
	if err := c.do(ctx, http.MethodGet, KeyGenerationPath, token, nil, &out); err != nil {
		return KeyGeneration{}, keyGenerationError(err)
	}
	if out.Generation < 0 {
		return KeyGeneration{}, fmt.Errorf("control plane answered with key generation %d", out.Generation)
	}
	return out, nil
}

// RaiseKeyGenerationFloor tells the control plane the account has reached
// generation. The floor moves up to it and never down, so no caller lowers
// it by reporting a smaller number — not a device whose view is behind, and
// not one about to be revoked. It answers with the floor as it stands.
//
// The client checks the answer is not below what it sent, which catches a
// control plane that is not keeping the number monotonic. That is a
// consistency check, not a way to compel one: a control plane is the party
// serving the floor and can answer whatever it likes. See the note above on
// what the floor is and is not worth.
func (c *Client) RaiseKeyGenerationFloor(ctx context.Context, token string, generation int) (KeyGeneration, error) {
	if generation < 1 {
		return KeyGeneration{}, fmt.Errorf("key generation %d is not a generation that can be reported", generation)
	}
	var out KeyGeneration
	in := map[string]int{"generation": generation}
	if err := c.do(ctx, http.MethodPost, KeyGenerationPath, token, in, &out); err != nil {
		return KeyGeneration{}, keyGenerationError(err)
	}
	if out.Generation < generation {
		return KeyGeneration{}, fmt.Errorf("control plane answered with key generation %d after being told %d, which is lower than the account has reached", out.Generation, generation)
	}
	return out, nil
}

func keyGenerationError(err error) error {
	var he *Error
	if !errors.As(err, &he) {
		return err
	}
	if he.Code == CodeKeyGenerationRange {
		return fmt.Errorf("%w: %s", ErrKeyGenerationRange, he.Message)
	}
	if he.Status == http.StatusNotFound {
		return ErrNoKeyGenerationFloor
	}
	if he.Status == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	return err
}
