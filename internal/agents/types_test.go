package agents

import (
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestCeilingAliasesMatchExistingBounds(t *testing.T) {
	t.Parallel()
	if MaxJSONLineBytes != sessionindex.MaxJSONLineBytes {
		t.Fatalf("MaxJSONLineBytes = %d, want sessionindex alias", MaxJSONLineBytes)
	}
	if MaxJSONLineBytes != 4<<20 {
		t.Fatalf("MaxJSONLineBytes = %d, want 4MiB", MaxJSONLineBytes)
	}
	if MaxSearchTextBytes != sessionindex.MaxSearchTextBytes {
		t.Fatalf("MaxSearchTextBytes = %d, want sessionindex alias", MaxSearchTextBytes)
	}
	if MaxSearchTextBytes != 256<<10 {
		t.Fatalf("MaxSearchTextBytes = %d, want 256KiB", MaxSearchTextBytes)
	}
	if MaxFileReferences != sessionindex.MaxFileReferences {
		t.Fatalf("MaxFileReferences = %d, want sessionindex alias", MaxFileReferences)
	}
	if MaxFileReferences != 512 {
		t.Fatalf("MaxFileReferences = %d, want 512", MaxFileReferences)
	}
	if DefaultMaxArgvBytes != handoff.DefaultMaxArgvBytes {
		t.Fatalf("DefaultMaxArgvBytes = %d, want handoff alias", DefaultMaxArgvBytes)
	}
	if DefaultMaxArgvBytes != 24<<10 {
		t.Fatalf("DefaultMaxArgvBytes = %d, want 24KiB", DefaultMaxArgvBytes)
	}
}

func TestTierString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tier Tier
		want string
	}{
		{TierKnown, "T0"},
		{TierDiscover, "T1"},
		{TierHandoffFrom, "T2"},
		{TierResume, "T3"},
		{TierHandoffTo, "T4"},
		{TierSync, "T5"},
		{Tier(99), "T?"},
	}
	for _, tt := range tests {
		if got := tt.tier.String(); got != tt.want {
			t.Fatalf("Tier(%d).String() = %q, want %q", tt.tier, got, tt.want)
		}
	}
}

func TestRootMatchesAndHomeDirJoin(t *testing.T) {
	t.Parallel()
	if !(Root{Path: "a"}).Matches("windows") {
		t.Fatal("empty OS should match every platform")
	}
	if (Root{Path: "a", OS: "macos"}).Matches("windows") {
		t.Fatal("macos root matched windows")
	}
	if got := HomeDir("/tmp/home").Join(".kimi", "sessions"); got == "" {
		t.Fatal("Join returned empty path")
	}
}

func TestNativeSpecArgvBudget(t *testing.T) {
	t.Parallel()
	if (*NativeSpec)(nil).ArgvBudget() != DefaultMaxArgvBytes {
		t.Fatal("nil spec should use default argv budget")
	}
	if (&NativeSpec{}).ArgvBudget() != DefaultMaxArgvBytes {
		t.Fatal("zero MaxArgvBytes should use default argv budget")
	}
	if (&NativeSpec{MaxArgvBytes: 1024}).ArgvBudget() != 1024 {
		t.Fatal("explicit argv budget was ignored")
	}
}

func TestEnvLookupAndHomeDir(t *testing.T) {
	t.Parallel()
	env := Env{
		Home: "/tmp/fixture-home",
		LookupEnv: func(key string) string {
			if key == "KIMI_CODE_HOME" {
				return "/tmp/kimi"
			}
			return ""
		},
		FixtureRoot: "/tmp/fixtures/kimi",
	}
	if got := env.Lookup("KIMI_CODE_HOME"); got != "/tmp/kimi" {
		t.Fatalf("Lookup = %q", got)
	}
	if got := env.Lookup("MISSING"); got != "" {
		t.Fatalf("missing lookup = %q", got)
	}
	home, err := env.HomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if home.String() != "/tmp/fixture-home" {
		t.Fatalf("HomeDir = %q", home)
	}
}

func TestClosedEnumerations(t *testing.T) {
	t.Parallel()
	reasons := []T0Reason{
		T0NoLocalHistory, T0ServerBacked, T0DesktopOnly,
		T0UnidentifiedProduct, T0UnofficialDistributionOnly, T0LayoutUnverified,
	}
	if len(reasons) != 6 {
		t.Fatalf("T0Reason count = %d", len(reasons))
	}
	if FamilyHomeTree != "F1" || FamilyCLIQuery != "F2" || FamilyEmbeddedDB != "F3" ||
		FamilyProjectFile != "F4" || FamilyRemote != "F5" {
		t.Fatal("storage family tokens drifted from the SDK")
	}
	if ProjectKeyNone != "none" || ProjectKeyPathSlug != "path_slug" ||
		ProjectKeyPathHash != "path_hash" || ProjectKeyURLEncoding != "url_encoding" ||
		ProjectKeyOpaqueID != "opaque_id" {
		t.Fatal("project key tokens drifted from the SDK")
	}
}
