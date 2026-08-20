package agentcheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// installedAgent builds a layout-recognized agent root with a resolvable
// executable, so only the version probe decides the outcome.
func installedAgent(t *testing.T, runner VersionRunner, timeout time.Duration) (string, Options) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root, Options{
		Root: root,
		LookPath: func(value string) (string, error) {
			return filepath.Join(root, "bin", value), nil
		},
		Runner:          runner,
		CaptureIdentity: testExecutableIdentity,
		Timeout:         timeout,
	}
}

// A version probe that only ran out of time has measured nothing. Reporting
// that as "no version" is indistinguishable from an uninstalled agent, and a
// caller gating on version would wave an out-of-range install straight through.
func TestInstalledVersionSeparatesFailedProbeFromAbsentVersion(t *testing.T) {
	t.Parallel()

	hang := versionRunnerFunc(func(ctx context.Context, _ string, _ ...string) (VersionOutput, error) {
		<-ctx.Done()
		return VersionOutput{}, ctx.Err()
	})
	_, opts := installedAgent(t, hang, 20*time.Millisecond)
	version, evidence := InstalledVersion(context.Background(), "claude", opts)
	if version != "" || evidence != VersionProbeFailed {
		t.Fatalf("timed-out probe = (%q, %q), want (\"\", %q)", version, evidence, VersionProbeFailed)
	}

	// A refusal to execute is equally a failed measurement.
	broken := versionRunnerFunc(func(context.Context, string, ...string) (VersionOutput, error) {
		return VersionOutput{}, errors.New("exec format error")
	})
	_, opts = installedAgent(t, broken, time.Second)
	if _, evidence := InstalledVersion(context.Background(), "claude", opts); evidence != VersionProbeFailed {
		t.Fatalf("failed exec = %q, want %q", evidence, VersionProbeFailed)
	}

	// Output that is not a version genuinely carries no version to read, and a
	// missing agent has nothing to probe. Both are absence, not failure.
	garbage := versionRunnerFunc(func(context.Context, string, ...string) (VersionOutput, error) {
		return VersionOutput{Stdout: "not a version\n"}, nil
	})
	_, opts = installedAgent(t, garbage, time.Second)
	if _, evidence := InstalledVersion(context.Background(), "claude", opts); evidence != VersionUnavailable {
		t.Fatalf("unparseable output = %q, want %q", evidence, VersionUnavailable)
	}

	_, opts = installedAgent(t, garbage, time.Second)
	opts.LookPath = func(string) (string, error) { return "", errors.New("not installed") }
	if _, evidence := InstalledVersion(context.Background(), "claude", opts); evidence != VersionUnavailable {
		t.Fatalf("uninstalled agent = %q, want %q", evidence, VersionUnavailable)
	}

	// And a version that reads cleanly is still just a version.
	good := versionRunnerFunc(func(context.Context, string, ...string) (VersionOutput, error) {
		return VersionOutput{Stdout: "2.1.220 (Claude Code)\n"}, nil
	})
	_, opts = installedAgent(t, good, time.Second)
	version, evidence = InstalledVersion(context.Background(), "claude", opts)
	if version != "2.1.220" || evidence != VersionDetermined {
		t.Fatalf("clean probe = (%q, %q)", version, evidence)
	}
}

// Agent CLIs are language runtimes whose startup can exceed a two-second budget
// on a loaded machine. One slow moment must not decide compatibility, so a
// timed-out probe is measured once more before it is called a failure.
func TestInspectRetriesOnlyTimedOutVersionProbes(t *testing.T) {
	t.Parallel()

	var slowAttempts atomic.Int32
	slowOnce := versionRunnerFunc(func(ctx context.Context, _ string, _ ...string) (VersionOutput, error) {
		if slowAttempts.Add(1) == 1 {
			<-ctx.Done()
			return VersionOutput{}, ctx.Err()
		}
		return VersionOutput{Stdout: "9.9.9 (Claude Code)\n"}, nil
	})
	_, opts := installedAgent(t, slowOnce, 250*time.Millisecond)
	version, evidence := InstalledVersion(context.Background(), "claude", opts)
	if slowAttempts.Load() != 2 {
		t.Fatalf("version probe attempts = %d, want 2", slowAttempts.Load())
	}
	// The retry recovered the real answer, and it is out of range — exactly the
	// value the fail-open path used to discard.
	if version != "9.9.9" || evidence != VersionDetermined {
		t.Fatalf("retried probe = (%q, %q)", version, evidence)
	}
	if SupportedVersion("claude", version) {
		t.Fatalf("fixture version %q is inside the verified range; pick one outside it", version)
	}

	// A deterministic failure gains nothing from a second run, and an
	// executable that changed underneath the probe must be reported rather than
	// re-rolled until it looks stable.
	var failAttempts atomic.Int32
	alwaysFails := versionRunnerFunc(func(context.Context, string, ...string) (VersionOutput, error) {
		failAttempts.Add(1)
		return VersionOutput{}, errors.New("exec format error")
	})
	_, opts = installedAgent(t, alwaysFails, time.Second)
	if _, evidence := InstalledVersion(context.Background(), "claude", opts); evidence != VersionProbeFailed {
		t.Fatalf("evidence = %q, want %q", evidence, VersionProbeFailed)
	}
	if failAttempts.Load() != 1 {
		t.Fatalf("non-timeout failure attempts = %d, want 1", failAttempts.Load())
	}
}

// A caller that cancels must not be answered with a retry it did not ask for.
func TestInspectDoesNotRetryAfterCallerCancellation(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	hang := versionRunnerFunc(func(ctx context.Context, _ string, _ ...string) (VersionOutput, error) {
		attempts.Add(1)
		<-ctx.Done()
		return VersionOutput{}, ctx.Err()
	})
	_, opts := installedAgent(t, hang, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	result := Inspect(ctx, "claude", opts)
	cancel()
	if result.Status != StatusError {
		t.Fatalf("cancelled probe status = %q, want %q", result.Status, StatusError)
	}
	if attempts.Load() != 1 {
		t.Fatalf("attempts after caller cancellation = %d, want 1", attempts.Load())
	}
}

// TestOutOfRangeVersionNamesTheRange covers Matrix E4, which requires that a
// version below the supported minimum and one above the maximum both refuse
// while naming the range. The refusal used to say only that the version was
// "outside the verified range", which leaves the user nothing to act on.
func TestOutOfRangeVersionNamesTheRange(t *testing.T) {
	t.Parallel()
	for _, version := range []string{"2.0.0", "9.9.9"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			runner := versionRunnerFunc(func(context.Context, string, ...string) (VersionOutput, error) {
				return VersionOutput{Stdout: version + " (Claude Code)\n"}, nil
			})
			_, opts := installedAgent(t, runner, time.Second)
			result := Inspect(context.Background(), "claude", opts)
			if result.Status == StatusSupported {
				t.Fatalf("version %s was accepted; it is outside the verified range", version)
			}
			for _, want := range []string{version, "2.1.219", "2.1.229"} {
				if !strings.Contains(result.Message, want) {
					t.Fatalf("message %q does not name %q", result.Message, want)
				}
			}
		})
	}
}

// TestVersionRangeRendersOpenBounds keeps an absent bound explicit rather than
// silently missing from a refusal.
func TestVersionRangeRendersOpenBounds(t *testing.T) {
	t.Parallel()
	tests := []struct{ min, max, want string }{
		{"1.0.0", "2.0.0", "1.0.0 to 2.0.0 inclusive"},
		{"1.0.0", "", "1.0.0 and newer"},
		{"", "2.0.0", "up to and including 2.0.0"},
		{"", "", "(no verified range is declared)"},
	}
	for _, tt := range tests {
		if got := renderVersionRange(tt.min, tt.max); got != tt.want {
			t.Fatalf("renderVersionRange(%q, %q) = %q, want %q", tt.min, tt.max, got, tt.want)
		}
	}
}
