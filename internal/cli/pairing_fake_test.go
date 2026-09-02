package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// --- pairing endpoints of the fake control plane ---

// fakePairing is one relayed device-approval request. The fake, like hopd,
// stores only opaque strings: it can neither check the binding nor open
// the payload.
type fakePairing struct {
	id, deviceID             string
	version                  int
	publicKey, salt, binding string
	status                   string
	payload                  string
	generation               int
	approvedBy               string
	expired                  bool
	claims                   int
	createdAt, expiresAt     time.Time
}

func (f *fakeControlPlane) registerPairing(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/pairing", f.createPairing)
	mux.HandleFunc("GET /v1/pairing", f.listPairings)
	mux.HandleFunc("POST /v1/pairing/{id}/approve", f.approvePairing)
	mux.HandleFunc("POST /v1/pairing/{id}/claim", f.claimPairing)
	mux.HandleFunc("POST /v1/pairing/{id}/expire", f.expirePairing)
	mux.HandleFunc("GET /v1/devices", f.listDevices)
	mux.HandleFunc("DELETE /v1/devices/{id}", f.revokeDevice)
	mux.HandleFunc("POST /v1/devices/{id}/revoke", f.revokeDevice)
	mux.HandleFunc("GET "+hop.KeyGenerationPath, f.readKeyGeneration)
	mux.HandleFunc("POST "+hop.KeyGenerationPath, f.raiseKeyGeneration)
}

// readKeyGeneration and raiseKeyGeneration are the account key-generation
// floor, in the shape hopd serves it (private docs/hop.md, "Key
// generation"): a per-account counter that only ever goes up, served to and
// raised by authenticated devices. A revoked device's token is gone from
// f.tokens, so identityFor answers 401 and it can neither read the floor
// nor raise it. A report at or below the floor is not an error; it answers
// with the floor as it stands and `raised: false`.
//
// Setting noKeyGenerationFloor makes the fake behave like a control plane
// that predates the route, which is the case the client falls back for.
func (f *fakeControlPlane) readKeyGeneration(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.identityFor(w, r); !ok {
		return
	}
	if f.noKeyGenerationFloor {
		writeFakeError(w, 404, "this control plane does not carry a key generation floor")
		return
	}
	f.keyGenerationReads++
	_ = json.NewEncoder(w).Encode(f.keyGenerationView(false))
}

func (f *fakeControlPlane) raiseKeyGeneration(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.identityFor(w, r)
	if !ok {
		return
	}
	if f.noKeyGenerationFloor {
		writeFakeError(w, 404, "this control plane does not carry a key generation floor")
		return
	}
	var req struct {
		Generation int `json:"generation"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	// hopd bounds the counter rather than verifying it; it holds no keyring
	// and cannot see whether a rollover happened.
	if req.Generation < 1 || req.Generation > fakeMaxKeyGeneration {
		writeFakeErrorCode(w, 400, hop.CodeKeyGenerationRange,
			fmt.Sprintf("generation must be a whole number between 1 and %d", fakeMaxKeyGeneration))
		return
	}
	raised := req.Generation > f.keyGeneration
	if raised {
		f.keyGeneration = req.Generation
		f.keyGenerationAt = time.Now().UTC().Format(time.RFC3339)
		f.keyGenerationBy = id.Device.ID
	}
	_ = json.NewEncoder(w).Encode(f.keyGenerationView(raised))
}

// fakeMaxKeyGeneration mirrors internal/store.MaxKeyGeneration in hopd.
const fakeMaxKeyGeneration = 1000

func (f *fakeControlPlane) keyGenerationView(raised bool) map[string]any {
	view := map[string]any{"generation": f.keyGeneration}
	if f.keyGenerationAt != "" {
		view["raised_at"] = f.keyGenerationAt
		view["raised_by"] = f.keyGenerationBy
	}
	if raised {
		view["raised"] = true
	}
	return view
}

// identityFor resolves the bearer token, answering the 401 itself.
func (f *fakeControlPlane) identityFor(w http.ResponseWriter, r *http.Request) (hop.Identity, bool) {
	id, ok := f.tokens[trimBearer(r)]
	if !ok {
		writeFakeError(w, 401, "unknown or revoked device token")
	}
	return id, ok
}

func trimBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 {
		return h[7:]
	}
	return ""
}

func (f *fakeControlPlane) pairingView(p *fakePairing) map[string]any {
	dev := f.deviceByID(p.deviceID)
	status := p.status
	if p.expired && status == "pending" {
		status = "expired"
	}
	return map[string]any{
		"id": p.id, "version": p.version, "status": status, "device": dev,
		"public_key": p.publicKey, "salt": p.salt, "binding": p.binding,
		"created_at": p.createdAt.Format(time.RFC3339Nano), "expires_at": p.expiresAt.Format(time.RFC3339Nano),
		"interval_seconds": 0,
	}
}

func (f *fakeControlPlane) deviceByID(id string) hop.Device {
	for _, identity := range f.tokens {
		if identity.Device.ID == id {
			return identity.Device
		}
	}
	if d, ok := f.revoked[id]; ok {
		return d
	}
	return hop.Device{ID: id}
}

// revokeDevice is DELETE /v1/devices/{id}: like hopd, the device record
// stays with revoked_at set, its token is forgotten so every later call
// (minting and pairing included) answers 401, its pending pairing request
// is expired, and a device_revoked event is recorded once.
func (f *fakeControlPlane) revokeDevice(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.identityFor(w, r)
	if !ok {
		return
	}
	target := r.PathValue("id")
	if target == id.Device.ID {
		writeFakeErrorCode(w, 400, "self_revoke", "a device cannot revoke itself; revoke it from another enrolled device")
		return
	}
	if d, ok := f.revoked[target]; ok {
		_ = json.NewEncoder(w).Encode(map[string]any{"device": d, "revoked": false})
		return
	}
	for token, identity := range f.tokens {
		if identity.Device.ID != target {
			continue
		}
		if identity.Account.ID != id.Account.ID {
			writeFakeErrorCode(w, 404, "device_unknown", "no such device on this account")
			return
		}
		d := identity.Device
		d.RevokedAt = time.Now().UTC().Format(time.RFC3339)
		if f.revoked == nil {
			f.revoked = map[string]hop.Device{}
		}
		f.revoked[target] = d
		delete(f.tokens, token)
		for _, p := range f.pairings {
			if p.deviceID == target && p.status == "pending" {
				p.status = "expired"
			}
		}
		f.events = append(f.events, "device_revoked:"+target)
		_ = json.NewEncoder(w).Encode(map[string]any{"device": d, "revoked": true})
		return
	}
	writeFakeErrorCode(w, 404, "device_unknown", "no such device on this account")
}

func (f *fakeControlPlane) createPairing(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.identityFor(w, r)
	if !ok {
		return
	}
	var req struct {
		Version   int    `json:"version"`
		PublicKey string `json:"public_key"`
		Salt      string `json:"salt"`
		Binding   string `json:"binding"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Version == 0 {
		req.Version = hop.PairingVersion1
	}
	if req.PublicKey == "" || req.Salt == "" || req.Binding == "" {
		writeFakeError(w, 400, "public_key, salt, and binding are required")
		return
	}
	if req.Version != hop.PairingVersion1 && req.Version != hop.PairingVersion2 {
		writeFakeError(w, 400, "version must be 1 or 2")
		return
	}
	// Like hopd: one open request per device.
	for _, p := range f.pairings {
		if p.deviceID == id.Device.ID && p.status == "pending" {
			p.status = "expired"
		}
	}
	f.pairingSeq++
	p := &fakePairing{
		id: "pair-" + strconv.Itoa(f.pairingSeq), deviceID: id.Device.ID, version: req.Version,
		publicKey: req.PublicKey, salt: req.Salt, binding: req.Binding, status: "pending",
		createdAt: time.Now().UTC(), expiresAt: time.Now().UTC().Add(10 * time.Minute),
	}
	if f.pairings == nil {
		f.pairings = map[string]*fakePairing{}
	}
	f.pairings[p.id] = p
	w.WriteHeader(201)
	_ = json.NewEncoder(w).Encode(f.pairingView(p))
}

func (f *fakeControlPlane) listPairings(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.identityFor(w, r); !ok {
		return
	}
	requests := []map[string]any{}
	for i := 1; i <= f.pairingSeq; i++ {
		if p, ok := f.pairings["pair-"+strconv.Itoa(i)]; ok && p.status == "pending" && !p.expired {
			requests = append(requests, f.pairingView(p))
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"requests": requests})
}

func (f *fakeControlPlane) pairingByPath(w http.ResponseWriter, r *http.Request) (*fakePairing, bool) {
	p, ok := f.pairings[r.PathValue("id")]
	if !ok {
		writeFakeError(w, 404, "unknown pairing request")
		return nil, false
	}
	return p, true
}

func (f *fakeControlPlane) approvePairing(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.identityFor(w, r)
	if !ok {
		return
	}
	p, ok := f.pairingByPath(w, r)
	if !ok {
		return
	}
	if p.deviceID == id.Device.ID {
		writeFakeError(w, 403, "a device cannot approve its own pairing request")
		return
	}
	var req struct {
		Payload       string `json:"payload"`
		KeyGeneration int    `json:"key_generation"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	switch {
	case p.expired:
		writeFakeErrorCode(w, 410, "pairing_expired", "this pairing request expired; run rein account join again on the new device")
	case p.status != "pending":
		writeFakeErrorCode(w, 409, "pairing_decided", "this pairing request was already approved, collected, or cancelled")
	case req.Payload == "" || req.KeyGeneration <= 0:
		writeFakeError(w, 400, "payload and key_generation are required")
	default:
		p.status, p.payload, p.generation, p.approvedBy = "approved", req.Payload, req.KeyGeneration, id.Device.ID
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "approved", "version": p.version})
	}
}

func (f *fakeControlPlane) claimPairing(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.identityFor(w, r)
	if !ok {
		return
	}
	p, ok := f.pairingByPath(w, r)
	if !ok {
		return
	}
	if p.deviceID != id.Device.ID {
		writeFakeError(w, 403, "only the device that opened this pairing request can collect it")
		return
	}
	p.claims++
	w.Header().Set("Content-Type", "application/json")
	switch {
	case p.expired || p.status == "expired":
		p.payload = ""
		w.WriteHeader(410)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "expired", "version": p.version})
	case p.status == "consumed":
		w.WriteHeader(410)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "consumed", "version": p.version})
	case p.status == "pending":
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "pending", "version": p.version})
	default:
		payload := p.payload
		p.status, p.payload = "consumed", ""
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "approved", "version": p.version, "payload": payload, "key_generation": p.generation,
			"approved_by": f.deviceByID(p.approvedBy),
		})
	}
}

func (f *fakeControlPlane) expirePairing(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.identityFor(w, r); !ok {
		return
	}
	p, ok := f.pairingByPath(w, r)
	if !ok {
		return
	}
	if p.status != "pending" {
		writeFakeErrorCode(w, 409, "pairing_decided", "this pairing request was already approved, collected, or cancelled")
		return
	}
	p.status = "expired"
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "expired"})
}

func (f *fakeControlPlane) listDevices(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.identityFor(w, r); !ok {
		return
	}
	devices := []hop.Device{}
	seen := map[string]bool{}
	for i := 1; i <= f.seq; i++ {
		id := "dev-sess-" + strconv.Itoa(i)
		if d, ok := f.revoked[id]; ok && !seen[id] {
			seen[id] = true
			devices = append(devices, d)
		}
		for _, identity := range f.tokens {
			if identity.Device.ID == id && !seen[id] {
				seen[id] = true
				devices = append(devices, identity.Device)
			}
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"devices": devices})
}
