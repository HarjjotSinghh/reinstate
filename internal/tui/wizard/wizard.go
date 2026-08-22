// Package wizard is the interactive front end for `rein init`.
//
// It collects the non-secret storage coordinates: which provider, which
// endpoint, which bucket, which prefix, and whether this device is joining an
// existing profile. Steps can be revisited, so a mistake on the last field does
// not discard the first.
//
// # Why secrets are not collected here
//
// The wizard deliberately stops before the access key and secret key. A Bubble
// Tea text input holds its value in an immutable Go string, which cannot be
// zeroed after use and may be copied by the runtime as the model is updated.
// The existing prompt reads secrets into a []byte that is wiped with
// crypto.Zero the moment it has been stored. Collecting credentials here would
// trade that guarantee for a nicer-looking prompt, which is a bad trade.
//
// So the wizard returns coordinates, the program exits and restores the
// terminal, and the caller reads credentials through the same hardened path it
// always has.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package wizard

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/tui"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// Provider is a storage backend preset. Presets exist because the endpoint
// format is the single most error-prone field in setup, and every provider
// documents it differently.
type Provider struct {
	Key      string
	Name     string
	Endpoint string // template, with <placeholder> segments the user fills in
	Region   string
	Note     string
}

// Providers are the presets offered, in the order they are shown.
var Providers = []Provider{
	{
		Key:      "r2",
		Name:     "Cloudflare R2",
		Endpoint: "https://<account-id>.r2.cloudflarestorage.com",
		Region:   "auto",
		Note:     "account ID is on the R2 overview page",
	},
	{
		Key:      "s3",
		Name:     "Amazon S3",
		Endpoint: "https://s3.<region>.amazonaws.com",
		Region:   "us-east-1",
		Note:     "use the regional endpoint for the bucket's region",
	},
	{
		Key:      "b2",
		Name:     "Backblaze B2",
		Endpoint: "https://s3.<region>.backblazeb2.com",
		Region:   "us-west-004",
		Note:     "the S3-compatible endpoint, not the native B2 one",
	},
	{
		Key:      "minio",
		Name:     "MinIO or self-hosted",
		Endpoint: "https://<host>:9000",
		Region:   "us-east-1",
		Note:     "any S3-compatible endpoint you operate",
	},
	{
		Key:      "other",
		Name:     "Other S3-compatible",
		Endpoint: "https://",
		Region:   "auto",
		Note:     "enter the endpoint exactly as your provider documents it",
	},
}

// Result is what the wizard collected. It contains no secret material.
type Result struct {
	Provider  string
	Endpoint  string
	Bucket    string
	Region    string
	Prefix    string
	ProfileID string
	// JoinExisting is true when this device is joining a profile created
	// elsewhere, which makes ProfileID required and changes the storage probe
	// into a manifest check.
	JoinExisting bool
}

// step is one screen of the wizard.
type step int

const (
	stepProvider step = iota
	stepEndpoint
	stepBucket
	stepRegion
	stepPrefix
	stepProfile
	stepProfileID
	stepReview
	stepCount
)

// title is the heading for a step.
func (s step) title() string {
	switch s {
	case stepProvider:
		return "Storage provider"
	case stepEndpoint:
		return "Endpoint"
	case stepBucket:
		return "Bucket"
	case stepRegion:
		return "Region"
	case stepPrefix:
		return "Key prefix"
	case stepProfile:
		return "This device"
	case stepProfileID:
		return "Profile ID"
	case stepReview:
		return "Review"
	default:
		return ""
	}
}

// Model is the wizard.
type Model struct {
	theme      ui.Theme
	capability ui.Capability

	step          step
	providerIndex int
	joinExisting  bool

	inputs map[step]*field

	width  int
	height int

	status    string
	confirmed bool
	quitting  bool
	err       error
}

// Options configure a wizard.
type Options struct {
	Theme      ui.Theme
	Capability ui.Capability
	// Defaults pre-fills fields, so rerunning after a failure does not mean
	// retyping everything.
	Defaults Result
}

// New builds a wizard.
func New(opts Options) *Model {
	model := &Model{
		theme:      opts.Theme,
		capability: opts.Capability,
		inputs:     make(map[step]*field, 5),
		width:      opts.Capability.Width,
		height:     opts.Capability.Height,
	}
	for index, provider := range Providers {
		if provider.Key == opts.Defaults.Provider {
			model.providerIndex = index
		}
	}
	model.joinExisting = opts.Defaults.JoinExisting

	for _, definition := range []struct {
		step        step
		value       string
		placeholder string
	}{
		{stepEndpoint, opts.Defaults.Endpoint, Providers[model.providerIndex].Endpoint},
		{stepBucket, opts.Defaults.Bucket, "reinstate"},
		{stepRegion, opts.Defaults.Region, Providers[model.providerIndex].Region},
		{stepPrefix, opts.Defaults.Prefix, "profiles/<profile-id>"},
		{stepProfileID, opts.Defaults.ProfileID, "paste the ID from your first device"},
	} {
		created := newField(definition.value, definition.placeholder)
		model.inputs[definition.step] = &created
	}
	return model
}

// Result returns what was collected. It is meaningful only once Confirmed.
func (m *Model) Result() Result {
	provider := Providers[m.providerIndex]
	region := strings.TrimSpace(m.value(stepRegion))
	if region == "" {
		region = provider.Region
	}
	return Result{
		Provider:     provider.Key,
		Endpoint:     strings.TrimSpace(m.value(stepEndpoint)),
		Bucket:       strings.TrimSpace(m.value(stepBucket)),
		Region:       region,
		Prefix:       strings.TrimSpace(m.value(stepPrefix)),
		ProfileID:    strings.TrimSpace(m.value(stepProfileID)),
		JoinExisting: m.joinExisting,
	}
}

// Confirmed reports whether the user completed the wizard.
func (m *Model) Confirmed() bool { return m.confirmed }

// Intent implements tui.Surface. The wizard configures storage rather than
// acting on a session, so it never produces an action; the caller reads
// Confirmed and Result instead.
func (m *Model) Intent() tui.Intent { return tui.Intent{} }

// Err implements tui.Surface.
func (m *Model) Err() error { return m.err }

// Init implements tea.Model. The wizard needs no startup work.
func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) value(s step) string {
	if input, ok := m.inputs[s]; ok {
		return input.Value()
	}
	return ""
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = typed.Width, typed.Height
		return m, nil

	case tea.KeyMsg:
		return m.updateKey(typed)
	}
	return m.forwardToInput(msg)
}

func (m *Model) updateKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.confirmed = false
		m.quitting = true
		return m, tea.Quit

	case tea.KeyShiftTab:
		m.back()
		return m, nil

	case tea.KeyEnter:
		return m.advance()

	case tea.KeyUp:
		if m.step == stepProvider {
			m.selectProvider(-1)
			return m, nil
		}
		if m.step == stepProfile {
			m.joinExisting = !m.joinExisting
			return m, nil
		}
		m.back()
		return m, nil

	case tea.KeyDown, tea.KeyTab:
		if m.step == stepProvider {
			m.selectProvider(1)
			return m, nil
		}
		if m.step == stepProfile {
			m.joinExisting = !m.joinExisting
			return m, nil
		}
		return m.advance()

	case tea.KeySpace:
		if m.step == stepProfile {
			m.joinExisting = !m.joinExisting
			return m, nil
		}
	}
	return m.forwardToInput(key)
}

func (m *Model) forwardToInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}
	if input, ok := m.inputs[m.step]; ok {
		input.Update(key)
	}
	return m, nil
}

// selectProvider changes the preset and re-seeds any field the user has not
// already edited, so switching from R2 to S3 updates the endpoint template
// without discarding a bucket name that was already typed.
func (m *Model) selectProvider(delta int) {
	previous := Providers[m.providerIndex]
	m.providerIndex = wrap(m.providerIndex+delta, len(Providers))
	current := Providers[m.providerIndex]

	endpoint := m.inputs[stepEndpoint]
	if endpoint.Empty() || endpoint.Value() == previous.Endpoint {
		endpoint.SetValue("")
	}
	endpoint.placeholder = current.Endpoint

	region := m.inputs[stepRegion]
	if region.Empty() || region.Value() == previous.Region {
		region.SetValue("")
	}
	region.placeholder = current.Region
	m.status = ""
}

func (m *Model) back() {
	for next := m.step - 1; next >= stepProvider; next-- {
		if m.skip(next) {
			continue
		}
		m.step = next
		m.status = ""
		return
	}
}

// advance validates the current step and moves forward, or completes.
func (m *Model) advance() (tea.Model, tea.Cmd) {
	if problem := m.validate(m.step); problem != "" {
		m.status = problem
		return m, nil
	}
	if m.step == stepReview {
		m.confirmed = true
		m.quitting = true
		return m, tea.Quit
	}
	for next := m.step + 1; next < stepCount; next++ {
		if m.skip(next) {
			continue
		}
		m.step = next
		m.status = ""
		return m, nil
	}
	m.confirmed = true
	m.quitting = true
	return m, tea.Quit
}

// skip reports whether a step does not apply to the current answers.
func (m *Model) skip(s step) bool {
	return s == stepProfileID && !m.joinExisting
}

// validate returns a human problem statement, or empty when the step is fine.
//
// Validating per step rather than at the end is the point: a bad endpoint is
// reported while the reader is still looking at the endpoint field, not after
// they have typed four more things and waited for a network probe.
func (m *Model) validate(s step) string {
	switch s {
	case stepEndpoint:
		endpoint := strings.TrimSpace(m.value(stepEndpoint))
		switch {
		case endpoint == "":
			return "an endpoint is required"
		case !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://"):
			return "the endpoint must start with https:// or http://"
		case strings.ContainsAny(endpoint, "<>"):
			return "replace the <placeholder> part of the endpoint with a real value"
		}
	case stepBucket:
		bucket := strings.TrimSpace(m.value(stepBucket))
		if bucket == "" {
			return "a bucket name is required"
		}
		if strings.ContainsAny(bucket, " /\\") {
			return "a bucket name cannot contain spaces or slashes"
		}
	case stepProfileID:
		id := strings.TrimSpace(m.value(stepProfileID))
		if id == "" {
			return "joining an existing profile requires its ID"
		}
		if !looksLikeUUID(id) {
			return "a profile ID is a UUID copied from your first device"
		}
	}
	return ""
}

// looksLikeUUID checks the shape only. The authoritative parse happens in the
// command, which is where a bad value must fail; this exists so the reader
// finds out immediately rather than after a storage round trip.
func looksLikeUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		switch index {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			isHex := (char >= '0' && char <= '9') ||
				(char >= 'a' && char <= 'f') ||
				(char >= 'A' && char <= 'F')
			if !isHex {
				return false
			}
		}
	}
	return true
}

func wrap(index, length int) int {
	if length <= 0 {
		return 0
	}
	return ((index % length) + length) % length
}
