package adapter

import (
	"context"
	"io"
	"testing"
)

type fakeAdapter struct{ name string }

func (f *fakeAdapter) Name() string { return f.name }
func (f *fakeAdapter) Detect(context.Context) (Install, Compatibility, error) {
	return Install{Agent: f.name}, CompatibilityNotInstalled, nil
}
func (f *fakeAdapter) Discover(context.Context, DiscoverOptions) ([]Session, error) {
	return nil, nil
}
func (f *fakeAdapter) PlanExport(context.Context, Session, ExportOptions) (ExportPlan, error) {
	return ExportPlan{}, nil
}
func (f *fakeAdapter) Export(context.Context, ExportPlan, io.Writer) error { return nil }
func (f *fakeAdapter) PlanRestore(context.Context, Snapshot, RestoreOptions) (RestorePlan, error) {
	return RestorePlan{}, nil
}
func (f *fakeAdapter) Restore(context.Context, RestorePlan, io.Reader) error { return nil }
func (f *fakeAdapter) Exclusions() []Exclusion {
	return []Exclusion{{Pattern: "**/auth.json", Reason: "credentials"}}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(&fakeAdapter{name: "claude"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(&fakeAdapter{name: "claude"}); err == nil {
		t.Fatal("expected duplicate error")
	}
	a, ok := r.Get("claude")
	if !ok || a.Name() != "claude" {
		t.Fatal("get")
	}
}

func TestCanRestore(t *testing.T) {
	if !CanRestore(CompatibilitySupported, false) {
		t.Fatal("supported")
	}
	if CanRestore(CompatibilityUntested, false) {
		t.Fatal("untested without override")
	}
	if !CanRestore(CompatibilityUntested, true) {
		t.Fatal("untested with override")
	}
	if CanRestore(CompatibilityUnsupported, true) {
		t.Fatal("unsupported")
	}
	if CanRestore(CompatibilityNotInstalled, false) {
		t.Fatal("not installed")
	}
}
