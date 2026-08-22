// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package wizard

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/tui"
	"github.com/HarjjotSinghh/reinstate/internal/tui/tuitest"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// TestMain pins the rendering environment. lipgloss resolves its colour profile
// from stdout exactly once, on the first render, so forcing NO_COLOR before any
// frame is drawn makes a golden frame identical whether the suite runs under
// `go test` (piped stdout) or as a bare test binary in a colour terminal.
func TestMain(m *testing.M) {
	if err := os.Setenv("NO_COLOR", "1"); err != nil {
		fmt.Fprintln(os.Stderr, "set NO_COLOR:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// sampleProfileID is the UUID a second device pastes in. Its shape is the only
// thing the wizard checks; the command parses it for real.
const sampleProfileID = "6f1d2a8e-4c3b-4f5a-9d2e-7b8c9a0d1e2f"

// completeDefaults is a wizard rerun after a failure: every coordinate already
// filled in, which is also the state that lets a test reach any step without
// typing its way there.
func completeDefaults() Result {
	return Result{
		Provider: "s3",
		Endpoint: "https://s3.us-east-1.amazonaws.com",
		Bucket:   "reinstate-sessions",
		Region:   "us-east-1",
		Prefix:   "profiles/" + sampleProfileID,
	}
}

// joiningDefaults is the same rerun on a second device.
func joiningDefaults() Result {
	defaults := completeDefaults()
	defaults.ProfileID = sampleProfileID
	defaults.JoinExisting = true
	return defaults
}

// config describes one wizard under test. The zero value is a 100x26
// monochrome wizard with nothing filled in.
type config struct {
	width, height int
	defaults      Result
}

func start(t *testing.T, cfg config) (*tuitest.Driver, *Model) {
	t.Helper()
	if cfg.width == 0 {
		cfg.width = 100
	}
	if cfg.height == 0 {
		cfg.height = 26
	}
	capability := ui.Capability{
		Mode:    ui.ModeFull,
		Color:   ui.ColorNone,
		Unicode: true,
		Width:   cfg.width,
		Height:  cfg.height,
	}
	model := New(Options{
		Theme:      ui.NewTheme(capability),
		Capability: capability,
		Defaults:   cfg.defaults,
	})
	return tuitest.New(t, model, cfg.width, cfg.height), model
}

// send applies named keys through the local key table, which covers the editing
// and navigation keys the shared harness does not name. A name that is not a
// key is inserted as text in one press, the way a paste arrives.
func send(t *testing.T, driver *tuitest.Driver, names ...string) {
	t.Helper()
	for _, name := range names {
		driver.Send(key(t, name))
	}
}

// advanceTo presses enter until the wizard reaches a step, failing rather than
// looping if the step is never reached.
func advanceTo(t *testing.T, driver *tuitest.Driver, model *Model, target step) {
	t.Helper()
	for attempt := 0; model.step != target; attempt++ {
		if attempt > int(stepCount) {
			t.Fatalf("step %d was never reached; stopped at %d with status %q",
				target, model.step, model.status)
		}
		before := model.step
		send(t, driver, "enter")
		if model.step == before {
			t.Fatalf("advancing from step %d was refused: %q", before, model.status)
		}
	}
}

// replace clears the focused field and types a new value rune by rune.
func replace(t *testing.T, driver *tuitest.Driver, value string) {
	t.Helper()
	send(t, driver, "ctrl+u")
	driver.Type(value)
}

// assertFrameWidth is the layout contract: no rendered line may be wider than
// the terminal, measured in display cells rather than bytes or runes.
func assertFrameWidth(t *testing.T, frame string, width int) {
	t.Helper()
	for index, line := range strings.Split(frame, "\n") {
		if got := ui.Width(line); got > width {
			t.Errorf("line %d is %d cells wide, terminal is %d\n%q", index+1, got, width, line)
		}
	}
}

// stepNames make a failure readable; the step type is an unexported int.
func stepName(s step) string {
	if title := s.title(); title != "" {
		return title
	}
	return fmt.Sprintf("step(%d)", int(s))
}

func stepNames(steps []step) []string {
	names := make([]string, 0, len(steps))
	for _, current := range steps {
		names = append(names, stepName(current))
	}
	return names
}

// TestStepOrderAndSkipping is the reason the wizard has a skip rule at all: a
// device creating its own profile has no profile ID to paste, so asking for one
// would be a dead end. The step must vanish in both directions, and the
// progress counter must not count a step nobody will see.
func TestStepOrderAndSkipping(t *testing.T) {
	tests := []struct {
		name     string
		defaults Result
		want     []step
	}{
		{
			name:     "creating a profile skips the profile ID",
			defaults: completeDefaults(),
			want: []step{
				stepProvider, stepEndpoint, stepBucket,
				stepRegion, stepPrefix, stepProfile, stepReview,
			},
		},
		{
			name:     "joining a profile visits the profile ID",
			defaults: joiningDefaults(),
			want: []step{
				stepProvider, stepEndpoint, stepBucket, stepRegion,
				stepPrefix, stepProfile, stepProfileID, stepReview,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driver, model := start(t, config{defaults: test.defaults})

			forward := []step{model.step}
			for attempt := 0; model.step != stepReview; attempt++ {
				if attempt > int(stepCount) {
					t.Fatalf("the review step was never reached; stopped at %s", stepName(model.step))
				}
				send(t, driver, "enter")
				forward = append(forward, model.step)
			}
			if got := stepNames(forward); strings.Join(got, ",") != strings.Join(stepNames(test.want), ",") {
				t.Fatalf("forward order = %v, want %v", got, stepNames(test.want))
			}

			// The counter names the position among the steps that apply, so the
			// total never changes as the reader walks the same path.
			for index, current := range test.want {
				model.step = current
				want := fmt.Sprintf("step %d of %d", index+1, len(test.want))
				if got := model.progress(); got != want {
					t.Fatalf("on %s progress = %q, want %q", stepName(current), got, want)
				}
			}
			model.step = stepReview

			backward := []step{model.step}
			for attempt := 0; model.step != stepProvider; attempt++ {
				if attempt > int(stepCount) {
					t.Fatalf("the first step was never reached; stopped at %s", stepName(model.step))
				}
				send(t, driver, "shift+tab")
				backward = append(backward, model.step)
			}
			reversed := make([]step, 0, len(test.want))
			for index := len(test.want) - 1; index >= 0; index-- {
				reversed = append(reversed, test.want[index])
			}
			if got := stepNames(backward); strings.Join(got, ",") != strings.Join(stepNames(reversed), ",") {
				t.Fatalf("backward order = %v, want %v", got, stepNames(reversed))
			}
		})
	}

	t.Run("shift+tab on the first step stays put", func(t *testing.T) {
		driver, model := start(t, config{})
		send(t, driver, "shift+tab", "shift+tab")
		if model.step != stepProvider {
			t.Fatalf("step = %s, want the first step", stepName(model.step))
		}
	})

	t.Run("toggling the device answer changes the total", func(t *testing.T) {
		driver, model := start(t, config{defaults: completeDefaults()})
		advanceTo(t, driver, model, stepProfile)

		if got := model.progress(); got != "step 6 of 7" {
			t.Fatalf("progress = %q, want %q", got, "step 6 of 7")
		}
		send(t, driver, "space")
		if !model.joinExisting {
			t.Fatal("space did not choose joining an existing profile")
		}
		if got := model.progress(); got != "step 6 of 8" {
			t.Fatalf("after choosing to join, progress = %q, want %q", got, "step 6 of 8")
		}
		send(t, driver, "space")
		if model.joinExisting {
			t.Fatal("space did not toggle back")
		}
		if got := model.progress(); got != "step 6 of 7" {
			t.Fatalf("progress = %q, want %q", got, "step 6 of 7")
		}
	})

	t.Run("up and down also answer the device question", func(t *testing.T) {
		for _, name := range []string{"up", "down", "tab"} {
			t.Run(name, func(t *testing.T) {
				driver, model := start(t, config{defaults: completeDefaults()})
				advanceTo(t, driver, model, stepProfile)
				send(t, driver, name)
				if !model.joinExisting {
					t.Fatalf("%q did not change the answer", name)
				}
				if model.step != stepProfile {
					t.Fatalf("%q left the step, now on %s", name, stepName(model.step))
				}
			})
		}
	})
}

// TestGoingBackKeepsWhatWasTyped is the promise the package comment makes: a
// mistake on the last field must not discard the first.
func TestGoingBackKeepsWhatWasTyped(t *testing.T) {
	driver, model := start(t, config{})

	send(t, driver, "enter") // past the provider list
	driver.Type("https://minio.example.com:9000")
	send(t, driver, "enter")
	driver.Type("team-sessions")
	send(t, driver, "enter")
	driver.Type("eu-central-1")

	send(t, driver, "shift+tab")
	if model.step != stepBucket {
		t.Fatalf("step = %s, want the bucket step", stepName(model.step))
	}
	if got := model.value(stepBucket); got != "team-sessions" {
		t.Fatalf("bucket = %q, want it preserved", got)
	}

	send(t, driver, "shift+tab")
	if model.step != stepEndpoint {
		t.Fatalf("step = %s, want the endpoint step", stepName(model.step))
	}
	if got := model.value(stepEndpoint); got != "https://minio.example.com:9000" {
		t.Fatalf("endpoint = %q, want it preserved", got)
	}
	if !strings.Contains(driver.View(), "https://minio.example.com:9000") {
		t.Errorf("the preserved endpoint is not visible in the frame:\n%s", driver.View())
	}

	// Editing after going back keeps the rest, and forward still holds it.
	send(t, driver, "backspace", "backspace", "backspace", "backspace")
	driver.Type("9001")
	send(t, driver, "enter")
	if got := model.value(stepBucket); got != "team-sessions" {
		t.Fatalf("bucket = %q after re-editing the endpoint, want it preserved", got)
	}
	send(t, driver, "enter")
	if got := model.value(stepRegion); got != "eu-central-1" {
		t.Fatalf("region = %q, want it preserved", got)
	}
	if got := model.value(stepEndpoint); got != "https://minio.example.com:9001" {
		t.Fatalf("endpoint = %q, want the edit applied", got)
	}
}

// TestCancelKeysDecideNothing checks the exits. A cancelled wizard must look
// exactly like never having run one, because the caller writes config on the
// strength of Confirmed alone.
func TestCancelKeysDecideNothing(t *testing.T) {
	for _, name := range []string{"esc", "ctrl+c"} {
		t.Run(name, func(t *testing.T) {
			driver, model := start(t, config{defaults: completeDefaults()})
			advanceTo(t, driver, model, stepReview)

			send(t, driver, name)

			if model.Confirmed() {
				t.Fatalf("%q confirmed the wizard", name)
			}
			if !model.quitting {
				t.Fatalf("%q did not quit the surface", name)
			}
			if frame := driver.View(); frame != "" {
				t.Fatalf("a quitting surface must leave no frame behind, got:\n%s", frame)
			}
			if model.Err() != nil {
				t.Fatalf("unexpected error: %v", model.Err())
			}
			assertZeroIntent(t, model.Intent())
		})
	}

	t.Run("cancelling on the first step decides nothing either", func(t *testing.T) {
		driver, model := start(t, config{})
		send(t, driver, "esc")
		if model.Confirmed() || !model.quitting {
			t.Fatalf("confirmed = %v quitting = %v, want a clean cancel",
				model.Confirmed(), model.quitting)
		}
	})
}

// assertZeroIntent checks a wizard decided nothing about a session. Intent
// carries a slice, so it cannot simply be compared against its zero value.
func assertZeroIntent(t *testing.T, intent tui.Intent) {
	t.Helper()
	if intent.Chosen() {
		t.Fatalf("intent = %+v, want no action", intent)
	}
	if intent.Action != tui.ActionNone || intent.Reference != "" ||
		intent.Destination != "" || intent.Policy != "" || len(intent.AcknowledgedWarnings) != 0 {
		t.Fatalf("intent = %+v, want the zero intent", intent)
	}
}

// TestValidationBlocksTheStepItBelongsTo is the whole argument for validating
// per step: the reader is told about a bad endpoint while looking at the
// endpoint field, not after four more answers and a network probe.
func TestValidationBlocksTheStepItBelongsTo(t *testing.T) {
	tests := []struct {
		name    string
		step    step
		joining bool
		typed   string
		// spaceKey presses a real space between typed and typedAfter, which is
		// how a terminal delivers one: as its own key type, not as a rune.
		spaceKey   bool
		typedAfter string
		want       string
		fixed      string
	}{
		{
			name:  "an empty endpoint",
			step:  stepEndpoint,
			want:  "an endpoint is required",
			fixed: "https://s3.us-east-1.amazonaws.com",
		},
		{
			name:  "an endpoint of only spaces",
			step:  stepEndpoint,
			typed: "   ",
			want:  "an endpoint is required",
			fixed: "https://s3.us-east-1.amazonaws.com",
		},
		{
			name:  "an endpoint with no scheme",
			step:  stepEndpoint,
			typed: "s3.us-east-1.amazonaws.com",
			want:  "the endpoint must start with https:// or http://",
			fixed: "https://s3.us-east-1.amazonaws.com",
		},
		{
			name:  "an endpoint with the wrong scheme",
			step:  stepEndpoint,
			typed: "ftp://s3.us-east-1.amazonaws.com",
			want:  "the endpoint must start with https:// or http://",
			fixed: "http://minio.internal:9000",
		},
		{
			name:  "an unreplaced template",
			step:  stepEndpoint,
			typed: "https://<account-id>.r2.cloudflarestorage.com",
			want:  "replace the <placeholder> part of the endpoint with a real value",
			fixed: "https://a1b2c3.r2.cloudflarestorage.com",
		},
		{
			name:  "a half-replaced template",
			step:  stepEndpoint,
			typed: "https://s3.<region>.amazonaws.com",
			want:  "replace the <placeholder> part of the endpoint with a real value",
			fixed: "https://s3.eu-west-2.amazonaws.com",
		},
		{
			name:  "an empty bucket",
			step:  stepBucket,
			want:  "a bucket name is required",
			fixed: "reinstate",
		},
		{
			name:  "a bucket of only spaces",
			step:  stepBucket,
			typed: "  ",
			want:  "a bucket name is required",
			fixed: "reinstate",
		},
		{
			name:       "a bucket with a space in it",
			step:       stepBucket,
			typed:      "my",
			spaceKey:   true,
			typedAfter: "bucket",
			want:       "a bucket name cannot contain spaces or slashes",
			fixed:      "my-bucket",
		},
		{
			name:  "a bucket with a slash",
			step:  stepBucket,
			typed: "bucket/prefix",
			want:  "a bucket name cannot contain spaces or slashes",
			fixed: "bucket",
		},
		{
			name:  "a bucket with a backslash",
			step:  stepBucket,
			typed: `bucket\prefix`,
			want:  "a bucket name cannot contain spaces or slashes",
			fixed: "bucket",
		},
		{
			name:    "an empty profile ID when joining",
			step:    stepProfileID,
			joining: true,
			want:    "joining an existing profile requires its ID",
			fixed:   sampleProfileID,
		},
		{
			name:    "a profile ID that is not a UUID",
			step:    stepProfileID,
			joining: true,
			typed:   "my-laptop",
			want:    "a profile ID is a UUID copied from your first device",
			fixed:   sampleProfileID,
		},
		{
			name:    "a truncated profile ID",
			step:    stepProfileID,
			joining: true,
			typed:   sampleProfileID[:35],
			want:    "a profile ID is a UUID copied from your first device",
			fixed:   sampleProfileID,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defaults := completeDefaults()
			if test.joining {
				defaults = joiningDefaults()
			}
			driver, model := start(t, config{defaults: defaults})
			advanceTo(t, driver, model, test.step)

			replace(t, driver, test.typed)
			if test.spaceKey {
				send(t, driver, "space")
			}
			driver.Type(test.typedAfter)
			send(t, driver, "enter")

			if model.step != test.step {
				t.Fatalf("the wizard advanced to %s with an invalid value", stepName(model.step))
			}
			if model.status != test.want {
				t.Fatalf("status = %q, want %q", model.status, test.want)
			}
			frame := driver.View()
			if !strings.Contains(frame, test.want) {
				t.Fatalf("the refusal %q is not visible in the frame:\n%s", test.want, frame)
			}
			if model.Confirmed() || model.quitting {
				t.Fatal("a refused step ended the wizard")
			}

			// The same step accepts a good value, and the complaint goes away.
			replace(t, driver, test.fixed)
			send(t, driver, "enter")
			if model.step == test.step {
				t.Fatalf("a valid value was still refused: %q", model.status)
			}
			if model.status != "" {
				t.Fatalf("status = %q after advancing, want it cleared", model.status)
			}
		})
	}

	t.Run("the region and the prefix accept anything, including nothing", func(t *testing.T) {
		for _, current := range []step{stepRegion, stepPrefix} {
			driver, model := start(t, config{defaults: completeDefaults()})
			advanceTo(t, driver, model, current)
			send(t, driver, "ctrl+u", "enter")
			if model.step == current {
				t.Fatalf("%s refused an empty value: %q", stepName(current), model.status)
			}
		}
	})

	t.Run("the profile ID is not validated when it is not asked for", func(t *testing.T) {
		defaults := completeDefaults()
		defaults.ProfileID = "not-a-uuid"
		driver, model := start(t, config{defaults: defaults})
		advanceTo(t, driver, model, stepReview)
		if model.status != "" {
			t.Fatalf("status = %q, want a clean walk to the review", model.status)
		}
	})
}

func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "canonical lower case", value: sampleProfileID, want: true},
		{name: "canonical upper case", value: strings.ToUpper(sampleProfileID), want: true},
		{name: "mixed case", value: "6F1d2A8e-4c3B-4f5A-9d2E-7b8C9a0D1e2F", want: true},
		{name: "all zeroes", value: "00000000-0000-0000-0000-000000000000", want: true},
		{name: "all f", value: "ffffffff-ffff-ffff-ffff-ffffffffffff", want: true},
		{name: "empty", value: "", want: false},
		{name: "too short", value: sampleProfileID[:35], want: false},
		{name: "too long", value: sampleProfileID + "0", want: false},
		{name: "no dashes", value: strings.ReplaceAll(sampleProfileID, "-", "") + "aaaa", want: false},
		{
			name:  "dashes in the wrong places",
			value: "6f1d2a8e4-c3b-4f5a-9d2e-7b8c9a0d1e2f",
			want:  false,
		},
		{
			name:  "a dash where a hex digit belongs",
			value: "6f1d2a8-e4c3b-4f5a-9d2e-7b8c9a0d1e2f",
			want:  false,
		},
		{
			name:  "a non-hex letter",
			value: "6f1d2a8g-4c3b-4f5a-9d2e-7b8c9a0d1e2f",
			want:  false,
		},
		{
			name:  "a space instead of a digit",
			value: "6f1d2a8 -4c3b-4f5a-9d2e-7b8c9a0d1e2f",
			want:  false,
		},
		{
			name:  "an underscore instead of a dash",
			value: "6f1d2a8e_4c3b-4f5a-9d2e-7b8c9a0d1e2f",
			want:  false,
		},
		{
			name:  "braced, as Windows tools print it",
			value: "{6f1d2a8e-4c3b-4f5a-9d2e-7b8c9a0d1e2}",
			want:  false,
		},
		{
			name:  "a wide digit that is 36 runes but not hex",
			value: strings.Repeat("６", 36),
			want:  false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := looksLikeUUID(test.value); got != test.want {
				t.Fatalf("looksLikeUUID(%q) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}

// TestProviderCycling covers the preset list. Presets exist to fix the
// endpoint, which is the most error-prone field in setup, so switching one must
// re-seed the templates without throwing away an answer the reader typed.
func TestProviderCycling(t *testing.T) {
	t.Run("down wraps past the end", func(t *testing.T) {
		driver, model := start(t, config{})
		for index := range Providers {
			if model.providerIndex != index {
				t.Fatalf("provider index = %d, want %d", model.providerIndex, index)
			}
			if got := Providers[model.providerIndex].Key; !strings.Contains(driver.View(), Providers[model.providerIndex].Name) {
				t.Errorf("provider %q is not visible in the frame", got)
			}
			send(t, driver, "down")
		}
		if model.providerIndex != 0 {
			t.Fatalf("provider index = %d after wrapping, want 0", model.providerIndex)
		}
	})

	t.Run("up wraps past the start", func(t *testing.T) {
		driver, model := start(t, config{})
		send(t, driver, "up")
		if want := len(Providers) - 1; model.providerIndex != want {
			t.Fatalf("provider index = %d, want %d", model.providerIndex, want)
		}
		for range Providers {
			send(t, driver, "up")
		}
		if want := len(Providers) - 1; model.providerIndex != want {
			t.Fatalf("provider index = %d after a full cycle, want %d", model.providerIndex, want)
		}
	})

	t.Run("tab cycles forward too", func(t *testing.T) {
		driver, model := start(t, config{})
		send(t, driver, "tab")
		if model.providerIndex != 1 {
			t.Fatalf("provider index = %d, want 1", model.providerIndex)
		}
		if model.step != stepProvider {
			t.Fatalf("tab left the provider step for %s", stepName(model.step))
		}
	})

	t.Run("the placeholders follow the preset", func(t *testing.T) {
		driver, model := start(t, config{})
		for index := range Providers {
			want := Providers[index]
			if model.providerIndex != index {
				t.Fatalf("provider index = %d, want %d", model.providerIndex, index)
			}
			if got := model.inputs[stepEndpoint].placeholder; got != want.Endpoint {
				t.Fatalf("%s: endpoint placeholder = %q, want %q", want.Key, got, want.Endpoint)
			}
			if got := model.inputs[stepRegion].placeholder; got != want.Region {
				t.Fatalf("%s: region placeholder = %q, want %q", want.Key, got, want.Region)
			}
			send(t, driver, "down")
		}
	})

	t.Run("the template is shown in the endpoint hint", func(t *testing.T) {
		driver, model := start(t, config{})
		send(t, driver, "down", "enter") // Amazon S3
		if model.step != stepEndpoint {
			t.Fatalf("step = %s, want the endpoint step", stepName(model.step))
		}
		frame := driver.View()
		if !strings.Contains(frame, Providers[1].Endpoint) {
			t.Fatalf("the S3 template is not in the frame:\n%s", frame)
		}
	})

	t.Run("an untouched template value is re-seeded", func(t *testing.T) {
		defaults := Result{Provider: "r2", Endpoint: Providers[0].Endpoint, Region: Providers[0].Region}
		driver, model := start(t, config{defaults: defaults})

		send(t, driver, "down")

		if got := model.value(stepEndpoint); got != "" {
			t.Fatalf("endpoint = %q, want the stale template cleared", got)
		}
		if got := model.value(stepRegion); got != "" {
			t.Fatalf("region = %q, want the stale default cleared", got)
		}
		if got := model.inputs[stepEndpoint].placeholder; got != Providers[1].Endpoint {
			t.Fatalf("endpoint placeholder = %q, want %q", got, Providers[1].Endpoint)
		}
	})

	t.Run("a value the reader typed survives the switch", func(t *testing.T) {
		driver, model := start(t, config{})
		send(t, driver, "enter")
		driver.Type("https://mine.example.com")
		send(t, driver, "enter")
		driver.Type("my-bucket")
		send(t, driver, "enter")
		driver.Type("eu-west-2")

		send(t, driver, "shift+tab", "shift+tab", "shift+tab")
		if model.step != stepProvider {
			t.Fatalf("step = %s, want the provider step", stepName(model.step))
		}
		send(t, driver, "down", "down")

		if got := model.value(stepEndpoint); got != "https://mine.example.com" {
			t.Fatalf("endpoint = %q, want the typed value kept", got)
		}
		if got := model.value(stepRegion); got != "eu-west-2" {
			t.Fatalf("region = %q, want the typed value kept", got)
		}
		if got := model.value(stepBucket); got != "my-bucket" {
			t.Fatalf("bucket = %q, want the typed value kept", got)
		}
		// The placeholders still move, so the hint describes the new preset.
		if got := model.inputs[stepEndpoint].placeholder; got != Providers[2].Endpoint {
			t.Fatalf("endpoint placeholder = %q, want %q", got, Providers[2].Endpoint)
		}
	})

	t.Run("switching clears a stale complaint", func(t *testing.T) {
		driver, model := start(t, config{})
		send(t, driver, "enter")
		driver.Type("nonsense")
		send(t, driver, "enter")
		if model.status == "" {
			t.Fatal("the bad endpoint was accepted")
		}
		send(t, driver, "shift+tab", "down")
		if model.status != "" {
			t.Fatalf("status = %q, want it cleared by the switch", model.status)
		}
	})

	t.Run("a default provider key selects that preset", func(t *testing.T) {
		for index, provider := range Providers {
			_, model := start(t, config{defaults: Result{Provider: provider.Key}})
			if model.providerIndex != index {
				t.Fatalf("provider %q selected index %d, want %d", provider.Key, model.providerIndex, index)
			}
		}
		_, model := start(t, config{defaults: Result{Provider: "nonexistent"}})
		if model.providerIndex != 0 {
			t.Fatalf("an unknown provider selected index %d, want the first preset", model.providerIndex)
		}
	})
}

func TestResult(t *testing.T) {
	t.Run("values are trimmed", func(t *testing.T) {
		driver, model := start(t, config{})
		send(t, driver, "enter")
		driver.Type("  https://minio.example.com:9000  ")
		send(t, driver, "enter")
		driver.Type("  team-sessions ")
		send(t, driver, "enter")
		driver.Type(" eu-central-1  ")
		send(t, driver, "enter")
		driver.Type("  profiles/shared  ")

		got := model.Result()
		want := Result{
			Provider: "r2",
			Endpoint: "https://minio.example.com:9000",
			Bucket:   "team-sessions",
			Region:   "eu-central-1",
			Prefix:   "profiles/shared",
		}
		if got != want {
			t.Fatalf("Result =\n  %+v\nwant\n  %+v", got, want)
		}
	})

	t.Run("a blank region falls back to the preset", func(t *testing.T) {
		for index, provider := range Providers {
			defaults := completeDefaults()
			defaults.Provider = provider.Key
			defaults.Region = ""
			_, model := start(t, config{defaults: defaults})
			if model.providerIndex != index {
				t.Fatalf("provider %q selected index %d", provider.Key, model.providerIndex)
			}
			if got := model.Result().Region; got != provider.Region {
				t.Fatalf("%s: region = %q, want the preset default %q", provider.Key, got, provider.Region)
			}
		}
	})

	t.Run("a region of only spaces also falls back", func(t *testing.T) {
		defaults := completeDefaults()
		defaults.Region = "   "
		_, model := start(t, config{defaults: defaults})
		if got := model.Result().Region; got != Providers[1].Region {
			t.Fatalf("region = %q, want %q", got, Providers[1].Region)
		}
	})

	t.Run("joining carries the profile ID", func(t *testing.T) {
		_, model := start(t, config{defaults: joiningDefaults()})
		got := model.Result()
		if !got.JoinExisting {
			t.Fatal("JoinExisting = false, want true")
		}
		if got.ProfileID != sampleProfileID {
			t.Fatalf("ProfileID = %q, want %q", got.ProfileID, sampleProfileID)
		}
	})

	t.Run("a first device carries no profile ID", func(t *testing.T) {
		_, model := start(t, config{defaults: completeDefaults()})
		got := model.Result()
		if got.JoinExisting {
			t.Fatal("JoinExisting = true, want false")
		}
		if got.ProfileID != "" {
			t.Fatalf("ProfileID = %q, want empty", got.ProfileID)
		}
	})

	// The wizard collects coordinates and stops. Credentials are read later,
	// through the hardened prompt, for the reason the package comment gives.
	t.Run("nothing in the result is secret", func(t *testing.T) {
		_, model := start(t, config{defaults: joiningDefaults()})
		result := model.Result()
		for _, value := range []string{
			result.Provider, result.Endpoint, result.Bucket,
			result.Region, result.Prefix, result.ProfileID,
		} {
			if value == "" {
				continue
			}
			lowered := strings.ToLower(value)
			for _, forbidden := range []string{"secret", "passphrase", "password", "access-key"} {
				if strings.Contains(lowered, forbidden) {
					t.Fatalf("the result carries %q in %q", forbidden, value)
				}
			}
		}
	})
}

// TestTheWholeWizard drives the surface the way a person does: every value
// typed, every choice made with a key, and the result read at the end.
func TestTheWholeWizard(t *testing.T) {
	t.Run("a first device", func(t *testing.T) {
		driver, model := start(t, config{})

		// Backblaze B2 is two below the first preset.
		send(t, driver, "down", "down")
		send(t, driver, "enter")
		driver.Type("  https://s3.us-west-004.backblazeb2.com ")
		send(t, driver, "enter")
		driver.Type("team-sessions")
		send(t, driver, "enter")
		// The region is left blank on purpose: the preset answers it.
		send(t, driver, "enter")
		driver.Type("profiles/shared")
		send(t, driver, "enter")

		if model.step != stepProfile {
			t.Fatalf("step = %s, want the device question", stepName(model.step))
		}
		send(t, driver, "enter")
		if model.step != stepReview {
			t.Fatalf("step = %s, want the review", stepName(model.step))
		}
		if got := model.progress(); got != "step 7 of 7" {
			t.Fatalf("progress = %q, want %q", got, "step 7 of 7")
		}

		frame := driver.View()
		for _, want := range []string{
			"Backblaze B2", "https://s3.us-west-004.backblazeb2.com",
			"team-sessions", "us-west-004", "profiles/shared",
			"first device", "passphrase",
		} {
			if !strings.Contains(frame, want) {
				t.Errorf("the review does not mention %q:\n%s", want, frame)
			}
		}

		send(t, driver, "enter")

		if !model.Confirmed() {
			t.Fatal("enter on the review did not confirm")
		}
		if !model.quitting {
			t.Fatal("a confirmed wizard should quit")
		}
		if got := driver.View(); got != "" {
			t.Fatalf("a quitting surface must leave no frame behind, got:\n%s", got)
		}
		want := Result{
			Provider: "b2",
			Endpoint: "https://s3.us-west-004.backblazeb2.com",
			Bucket:   "team-sessions",
			Region:   "us-west-004",
			Prefix:   "profiles/shared",
		}
		if got := model.Result(); got != want {
			t.Fatalf("Result =\n  %+v\nwant\n  %+v", got, want)
		}
		if model.Err() != nil {
			t.Fatalf("unexpected error: %v", model.Err())
		}
	})

	t.Run("a second device joining a profile", func(t *testing.T) {
		driver, model := start(t, config{})

		send(t, driver, "enter") // Cloudflare R2
		driver.Type("https://a1b2c3d4.r2.cloudflarestorage.com")
		send(t, driver, "enter")
		driver.Type("reinstate")
		send(t, driver, "enter")
		driver.Type("auto")
		send(t, driver, "enter")
		// The prefix is left blank: it is derived from the profile ID.
		send(t, driver, "enter")
		send(t, driver, "down") // join a profile from another device
		send(t, driver, "enter")

		if model.step != stepProfileID {
			t.Fatalf("step = %s, want the profile ID step", stepName(model.step))
		}
		if got := model.progress(); got != "step 7 of 8" {
			t.Fatalf("progress = %q, want %q", got, "step 7 of 8")
		}
		driver.Type(sampleProfileID)
		send(t, driver, "enter")

		if model.step != stepReview {
			t.Fatalf("step = %s, want the review", stepName(model.step))
		}
		frame := driver.View()
		for _, want := range []string{"Cloudflare R2", "joining profile", sampleProfileID, "generated"} {
			if !strings.Contains(frame, want) {
				t.Errorf("the review does not mention %q:\n%s", want, frame)
			}
		}

		send(t, driver, "enter")

		want := Result{
			Provider:     "r2",
			Endpoint:     "https://a1b2c3d4.r2.cloudflarestorage.com",
			Bucket:       "reinstate",
			Region:       "auto",
			ProfileID:    sampleProfileID,
			JoinExisting: true,
		}
		if got := model.Result(); got != want {
			t.Fatalf("Result =\n  %+v\nwant\n  %+v", got, want)
		}
		if !model.Confirmed() {
			t.Fatal("enter on the review did not confirm")
		}
	})

	t.Run("down also advances a text step", func(t *testing.T) {
		driver, model := start(t, config{defaults: completeDefaults()})
		advanceTo(t, driver, model, stepEndpoint)
		send(t, driver, "down")
		if model.step != stepBucket {
			t.Fatalf("step = %s, want the bucket step", stepName(model.step))
		}
		send(t, driver, "up")
		if model.step != stepEndpoint {
			t.Fatalf("step = %s, want the endpoint step", stepName(model.step))
		}
	})
}

// TestIntentIsAlwaysEmpty is the boundary between configuring storage and
// acting on a session. The wizard implements tui.Surface, so a caller could
// read Intent from it; it must never carry an action, in any state, or a setup
// run could resume or fork something.
func TestIntentIsAlwaysEmpty(t *testing.T) {
	states := []struct {
		name string
		keys []string
	}{
		{name: "fresh"},
		{name: "provider chosen", keys: []string{"down", "enter"}},
		{name: "mid-form", keys: []string{"enter", "enter", "enter"}},
		{name: "refused", keys: []string{"enter", "ctrl+u", "enter"}},
		{name: "at the review", keys: []string{"enter", "enter", "enter", "enter", "enter", "enter"}},
		{name: "confirmed", keys: []string{"enter", "enter", "enter", "enter", "enter", "enter", "enter"}},
		{name: "cancelled", keys: []string{"enter", "esc"}},
	}
	for _, state := range states {
		t.Run(state.name, func(t *testing.T) {
			driver, model := start(t, config{defaults: completeDefaults()})
			send(t, driver, state.keys...)
			assertZeroIntent(t, model.Intent())
		})
	}
}

func TestUnhandledMessagesAreInert(t *testing.T) {
	driver, model := start(t, config{defaults: completeDefaults()})
	before := driver.View()

	driver.Send(struct{ unknown int }{})
	driver.Send(tea.KeyMsg{Type: tea.KeyF1})

	if got := driver.View(); got != before {
		t.Error("an unhandled message changed the frame")
	}
	if model.quitting || model.Confirmed() {
		t.Fatal("an unhandled message decided something")
	}
}

func TestResizeRelaysOut(t *testing.T) {
	driver, model := start(t, config{width: 120, height: 40, defaults: completeDefaults()})
	advanceTo(t, driver, model, stepReview)

	driver.Resize(60, 20)

	if model.width != 60 || model.height != 20 {
		t.Fatalf("size = %dx%d, want 60x20", model.width, model.height)
	}
	assertFrameWidth(t, driver.View(), 60)
}

// goldenSizes are the terminal sizes the golden frames pin, and the sizes the
// layout invariants are measured at.
var goldenSizes = []struct{ width, height int }{
	{100, 26},
	{100, 20},
	{80, 24},
	{60, 20},
}

// wizardStates walks the surface through every screen it has, including the
// refused ones, so the invariants below are measured on all of them.
var wizardStates = []struct {
	name     string
	defaults Result
	keys     []string
}{
	{name: "provider"},
	{name: "provider cycled", keys: []string{"down", "down", "down"}},
	{name: "endpoint", defaults: completeDefaults(), keys: []string{"enter"}},
	{name: "endpoint refused", defaults: completeDefaults(), keys: []string{"enter", "ctrl+u", "enter"}},
	{name: "bucket", defaults: completeDefaults(), keys: []string{"enter", "enter"}},
	{name: "bucket refused", defaults: completeDefaults(), keys: []string{"enter", "enter", "ctrl+u", "enter"}},
	{name: "region", defaults: completeDefaults(), keys: []string{"enter", "enter", "enter"}},
	{name: "prefix", defaults: completeDefaults(), keys: []string{"enter", "enter", "enter", "enter"}},
	{name: "device", defaults: completeDefaults(), keys: []string{"enter", "enter", "enter", "enter", "enter"}},
	{
		name:     "profile ID",
		defaults: joiningDefaults(),
		keys:     []string{"enter", "enter", "enter", "enter", "enter", "enter"},
	},
	{
		name:     "profile ID refused",
		defaults: joiningDefaults(),
		keys:     []string{"enter", "enter", "enter", "enter", "enter", "enter", "ctrl+u", "enter"},
	},
	{
		name:     "review",
		defaults: completeDefaults(),
		keys:     []string{"enter", "enter", "enter", "enter", "enter", "enter"},
	},
	{
		name:     "review joining",
		defaults: joiningDefaults(),
		keys:     []string{"enter", "enter", "enter", "enter", "enter", "enter", "enter"},
	},
}

// TestFrameFitsTheTerminal is the layout regression net: every screen, at every
// supported size, measured in display cells.
func TestFrameFitsTheTerminal(t *testing.T) {
	for _, size := range goldenSizes {
		for _, state := range wizardStates {
			t.Run(fmt.Sprintf("%dx%d/%s", size.width, size.height, state.name), func(t *testing.T) {
				driver, _ := start(t, config{
					width:    size.width,
					height:   size.height,
					defaults: state.defaults,
				})
				send(t, driver, state.keys...)

				frame := driver.View()
				if frame == "" {
					t.Fatal("the surface stopped rendering; this assertion would be vacuous")
				}
				assertFrameWidth(t, frame, size.width)
				if lines := strings.Count(frame, "\n") + 1; lines > size.height {
					t.Errorf("frame has %d lines, terminal has %d", lines, size.height)
				}
			})
		}
	}
}

// TestNoFrameCarriesAnEscapeSequence is a security invariant, not a cosmetic
// one. A default reaches this wizard from a pairing code that someone pasted
// out of a chat window, so an escape sequence surviving into a frame would let
// whoever wrote that code drive the reader's terminal.
func TestNoFrameCarriesAnEscapeSequence(t *testing.T) {
	hostile := Result{
		Provider:  "other",
		Endpoint:  "https://\x1b[31mevil\x1b[0m.example.com\x1b]0;pwned\a",
		Bucket:    "bucket\x1b[2J\x1b[H",
		Region:    "auto\nus-east-1",
		Prefix:    "profiles/\x1b[5m",
		ProfileID: sampleProfileID + "\x1b[1m",
	}
	for _, size := range goldenSizes {
		for _, state := range wizardStates {
			// Each state keeps its own device answer, so its key sequence still
			// lands on the screen it names. A hostile value may be refused on
			// the way — a refusal screen is a frame too, and is included here.
			defaults := hostile
			defaults.JoinExisting = state.defaults.JoinExisting
			name := fmt.Sprintf("%dx%d/%s", size.width, size.height, state.name)
			t.Run(name, func(t *testing.T) {
				driver, model := start(t, config{
					width:    size.width,
					height:   size.height,
					defaults: defaults,
				})
				send(t, driver, state.keys...)

				frame := driver.View()
				if frame == "" {
					t.Fatal("the surface stopped rendering; this assertion would be vacuous")
				}
				if strings.ContainsRune(frame, 0x1b) {
					t.Fatalf("frame contains a raw escape byte:\n%q", frame)
				}
				assertFrameWidth(t, frame, size.width)

				// The escaped text is still readable, just inert, and the value
				// the caller would write to config is inert too.
				if size.width >= 100 && state.name == "review" && !strings.Contains(frame, "[31mevil") {
					t.Errorf("the sanitized endpoint is unreadable:\n%s", frame)
				}
				result := model.Result()
				for _, value := range []string{
					result.Endpoint, result.Bucket, result.Region, result.Prefix, result.ProfileID,
				} {
					if strings.ContainsAny(value, "\x1b\n\r\t\x00") {
						t.Fatalf("Result carries a control character: %q", value)
					}
				}
			})
		}
	}
}

// TestGoldenFrames pins what a user actually sees. Regenerate with
// `go test ./internal/tui/wizard/ -update-golden` and review the diff: an
// unexplained change to one of these is a regression in the surface.
func TestGoldenFrames(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config
		keys   []string
		golden string
	}{
		{
			name:   "provider 100x26",
			cfg:    config{width: 100, height: 26},
			golden: "wizard_provider_100x26",
		},
		{
			name:   "endpoint refused 100x20",
			cfg:    config{width: 100, height: 20, defaults: completeDefaults()},
			keys:   []string{"enter", "ctrl+u", "s3.us-east-1.amazonaws.com", "enter"},
			golden: "wizard_endpoint_error_100x20",
		},
		{
			name:   "device 100x20",
			cfg:    config{width: 100, height: 20, defaults: completeDefaults()},
			keys:   []string{"enter", "enter", "enter", "enter", "enter"},
			golden: "wizard_profile_100x20",
		},
		{
			name:   "review joining 100x26",
			cfg:    config{width: 100, height: 26, defaults: joiningDefaults()},
			keys:   []string{"enter", "enter", "enter", "enter", "enter", "enter", "enter"},
			golden: "wizard_review_100x26",
		},
		{
			name:   "narrow review 60x20",
			cfg:    config{width: 60, height: 20, defaults: completeDefaults()},
			keys:   []string{"enter", "enter", "enter", "enter", "enter", "enter"},
			golden: "wizard_narrow_60x20",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driver, _ := start(t, test.cfg)
			send(t, driver, test.keys...)
			frame := driver.View()

			assertFrameWidth(t, frame, test.cfg.width)
			if lines := strings.Count(frame, "\n") + 1; lines > test.cfg.height {
				t.Errorf("frame has %d lines, terminal has %d", lines, test.cfg.height)
			}
			if strings.ContainsRune(frame, 0x1b) {
				t.Fatal("a golden frame must contain no escape sequences")
			}
			tuitest.AssertGolden(t, test.golden, frame)
		})
	}
}
