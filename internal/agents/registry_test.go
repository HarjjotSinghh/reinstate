package agents

import (
	"fmt"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func TestRegistryAccessorsAreSortedAndFiltered(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	MustRegister(testDescriptor("zeta", TierKnown, T0LayoutUnverified))
	MustRegister(testDescriptor("alpha", TierSync, ""))
	MustRegister(func() Descriptor {
		d := testDescriptor("mu", TierHandoffFrom, "")
		d.NewIndexSource = func(Env) (sessionindex.Source, error) { return nil, nil }
		d.NewReader = func(Env) (transcript.Reader, error) { return nil, nil }
		return d
	}())
	MustRegister(func() Descriptor {
		d := testDescriptor("beta", TierResume, "")
		d.Native = &NativeSpec{Executable: "beta"}
		d.NewIndexSource = func(Env) (sessionindex.Source, error) { return nil, nil }
		d.NewReader = func(Env) (transcript.Reader, error) { return nil, nil }
		return d
	}())

	if got, want := Keys(), []string{"alpha", "beta", "mu", "zeta"}; !stringSlicesEqual(got, want) {
		t.Fatalf("Keys() = %v, want %v", got, want)
	}
	all := All()
	if len(all) != 4 || all[0].Key != "alpha" || all[3].Key != "zeta" {
		t.Fatalf("All() order = %v", keysOf(all))
	}
	got, ok := Get("mu")
	if !ok || got.Tier != TierHandoffFrom {
		t.Fatalf("Get(mu) = %+v, %t", got, ok)
	}
	if _, ok := Get("missing"); ok {
		t.Fatal("Get(missing) succeeded")
	}

	if got := keysOf(AtLeast(TierResume)); !stringSlicesEqual(got, []string{"alpha", "beta"}) {
		t.Fatalf("AtLeast(T3) = %v", got)
	}
	if got := keysOf(Capable(CapabilityIndex)); !stringSlicesEqual(got, []string{"beta", "mu"}) {
		t.Fatalf("Capable(index) = %v", got)
	}
	if got := keysOf(Capable(CapabilityHandoffFrom)); !stringSlicesEqual(got, []string{"beta", "mu"}) {
		t.Fatalf("Capable(handoff_from) = %v", got)
	}
	if got := keysOf(Capable(CapabilityResume)); !stringSlicesEqual(got, []string{"beta"}) {
		t.Fatalf("Capable(resume) = %v", got)
	}
	if got := keysOf(Capable(CapabilitySync)); len(got) != 0 {
		t.Fatalf("Capable(sync) = %v", got)
	}
}

func TestMustRegisterPanics(t *testing.T) {
	tests := []struct {
		name    string
		desc    Descriptor
		want    string
		prepare func()
	}{
		{name: "empty key", desc: Descriptor{Tier: TierKnown, T0Reason: T0DesktopOnly}, want: "empty catalog key"},
		{name: "whitespace key", desc: Descriptor{Key: "  ", Tier: TierKnown, T0Reason: T0DesktopOnly}, want: "empty catalog key"},
		{
			name: "duplicate key",
			desc: testDescriptor("dup", TierKnown, T0ServerBacked),
			want: `duplicate catalog key "dup"`,
			prepare: func() {
				MustRegister(testDescriptor("dup", TierKnown, T0ServerBacked))
			},
		},
		{name: "T0 missing reason", desc: testDescriptor("t0", TierKnown, ""), want: "T0Reason required at T0"},
		{
			name: "reason above T0",
			desc: testDescriptor("t1", TierDiscover, T0LayoutUnverified),
			want: "T0Reason",
		},
		{
			name: "index constructor at T0",
			desc: func() Descriptor {
				d := testDescriptor("t0-index", TierKnown, T0NoLocalHistory)
				d.NewIndexSource = func(Env) (sessionindex.Source, error) { return nil, nil }
				return d
			}(),
			want: "NewIndexSource",
		},
		{
			name: "reader constructor at T1",
			desc: func() Descriptor {
				d := testDescriptor("t1-reader", TierDiscover, "")
				d.NewReader = func(Env) (transcript.Reader, error) { return nil, nil }
				return d
			}(),
			want: "NewReader",
		},
		{
			name: "target constructor at T3",
			desc: func() Descriptor {
				d := testDescriptor("t3-target", TierResume, "")
				d.NewTarget = func(Env) (handoff.HandoffTarget, error) { return nil, nil }
				return d
			}(),
			want: "NewTarget",
		},
		{
			name: "sync constructor at T4",
			desc: func() Descriptor {
				d := testDescriptor("t4-sync", TierHandoffTo, "")
				d.NewSyncAdapter = func(Env) (adapter.Adapter, error) { return nil, nil }
				return d
			}(),
			want: "NewSyncAdapter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetForTest()
			t.Cleanup(resetForTest)
			if tt.prepare != nil {
				tt.prepare()
			}
			defer func() {
				recovered := recover()
				if recovered == nil {
					t.Fatal("MustRegister did not panic")
				}
				msg := fmt.Sprint(recovered)
				if !strings.Contains(msg, tt.want) {
					t.Fatalf("panic %q, want substring %q", msg, tt.want)
				}
			}()
			MustRegister(tt.desc)
		})
	}
}

func TestMustRegisterAcceptsConstructorsAtDeclaredTier(t *testing.T) {
	resetForTest()
	t.Cleanup(resetForTest)

	d := testDescriptor("full", TierSync, "")
	d.Native = &NativeSpec{Executable: "full"}
	d.Version = &VersionSpec{Args: []string{"--version"}}
	d.NewIndexSource = func(Env) (sessionindex.Source, error) { return nil, nil }
	d.NewReader = func(Env) (transcript.Reader, error) { return nil, nil }
	d.NewTarget = func(Env) (handoff.HandoffTarget, error) { return nil, nil }
	d.NewSyncAdapter = func(Env) (adapter.Adapter, error) { return nil, nil }
	MustRegister(d)

	got, ok := Get("full")
	if !ok || !got.hasCapability(CapabilitySync) || !got.hasCapability(CapabilityHandoffTo) {
		t.Fatalf("registered T5 descriptor missing constructors: %+v", got)
	}
}

func testDescriptor(key string, tier Tier, reason T0Reason) Descriptor {
	return Descriptor{
		Key:         key,
		DisplayName: key,
		Vendor:      "test",
		Tier:        tier,
		Family:      FamilyHomeTree,
		T0Reason:    reason,
	}
}

func keysOf(descriptors []Descriptor) []string {
	keys := make([]string, len(descriptors))
	for i, d := range descriptors {
		keys[i] = d.Key
	}
	return keys
}

func stringSlicesEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
