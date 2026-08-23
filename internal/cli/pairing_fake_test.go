package cli

import (
	"encoding/json"
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
		"id": p.id, "status": status, "device": dev,
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
	return hop.Device{ID: id}
}

func (f *fakeControlPlane) createPairing(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.identityFor(w, r)
	if !ok {
		return
	}
	var req struct{ PublicKey, Salt, Binding string }
	var raw map[string]string
	_ = json.NewDecoder(r.Body).Decode(&raw)
	req.PublicKey, req.Salt, req.Binding = raw["public_key"], raw["salt"], raw["binding"]
	if req.PublicKey == "" || req.Salt == "" || req.Binding == "" {
		writeFakeError(w, 400, "public_key, salt, and binding are required")
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
		id: "pair-" + strconv.Itoa(f.pairingSeq), deviceID: id.Device.ID,
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
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
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
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "expired"})
	case p.status == "consumed":
		w.WriteHeader(410)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "consumed"})
	case p.status == "pending":
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "pending"})
	default:
		payload := p.payload
		p.status, p.payload = "consumed", ""
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "approved", "payload": payload, "key_generation": p.generation,
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
		for _, identity := range f.tokens {
			if identity.Device.ID == "dev-sess-"+strconv.Itoa(i) && !seen[identity.Device.ID] {
				seen[identity.Device.ID] = true
				devices = append(devices, identity.Device)
			}
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"devices": devices})
}
