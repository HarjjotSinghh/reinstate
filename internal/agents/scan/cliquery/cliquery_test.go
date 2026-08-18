package cliquery

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRunAppliesTimeoutAndOutputCeiling(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		var (
			gotName     string
			gotArgs     []string
			hadDeadline bool
		)
		output, err := Run(context.Background(), "opencode", []string{"session", "list", "--format", "json"}, Config{
			Timeout: 50 * time.Millisecond,
			Runner: RunnerFunc(func(ctx context.Context, name string, args ...string) ([]byte, error) {
				gotName = name
				gotArgs = append([]string(nil), args...)
				_, hadDeadline = ctx.Deadline()
				return []byte(`[{"id":"1"}]`), nil
			}),
		})
		if err != nil {
			t.Fatal(err)
		}
		if gotName != "opencode" || strings.Join(gotArgs, " ") != "session list --format json" {
			t.Fatalf("command = %q %v", gotName, gotArgs)
		}
		if !hadDeadline {
			t.Fatal("runner context had no deadline")
		}
		if string(output) != `[{"id":"1"}]` {
			t.Fatalf("output = %s", output)
		}
	})

	t.Run("non-zero exit", func(t *testing.T) {
		t.Parallel()
		want := exitError{code: 2}
		_, err := Run(context.Background(), "opencode", []string{"session", "list"}, Config{
			Runner: RunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
				return nil, want
			}),
		})
		if !errors.Is(err, want) {
			t.Fatalf("error = %v, want %v", err, want)
		}
	})

	t.Run("output ceiling", func(t *testing.T) {
		t.Parallel()
		_, err := Run(context.Background(), "opencode", nil, Config{
			MaxOutput: 8,
			Runner: RunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
				return []byte("0123456789"), nil
			}),
		})
		if !errors.Is(err, ErrOutputTooLarge) {
			t.Fatalf("error = %v, want %v", err, ErrOutputTooLarge)
		}
	})

	t.Run("caps requested ceiling at MaxJSONLineBytes", func(t *testing.T) {
		t.Parallel()
		_, err := Run(context.Background(), "opencode", nil, Config{
			MaxOutput: MaxJSONLineBytes + 64,
			Runner: RunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
				return make([]byte, MaxJSONLineBytes+1), nil
			}),
		})
		if !errors.Is(err, ErrOutputTooLarge) {
			t.Fatalf("error = %v, want %v", err, ErrOutputTooLarge)
		}
	})
}

func TestBoundedOutputStopsRetainingPastCeiling(t *testing.T) {
	t.Parallel()
	var out boundedOutput
	out.remaining = 4
	n, err := out.Write([]byte("abcdef"))
	if err != nil || n != 6 {
		t.Fatalf("Write() = %d, %v", n, err)
	}
	if !out.exceeded || out.String() != "abcd" {
		t.Fatalf("buffer = %q exceeded=%t", out.String(), out.exceeded)
	}
}

func TestDecodeJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload string
		wantID  string
		wantErr bool
	}{
		{name: "object", payload: `{"id":"oc-1","title":"listed"}`, wantID: "oc-1"},
		{name: "invalid", payload: `{`, wantErr: true},
		{name: "oversized", payload: `{"x":"` + strings.Repeat("a", MaxJSONLineBytes) + `"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var dest map[string]any
			err := DecodeJSON([]byte(tt.payload), &dest)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DecodeJSON() error = nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if dest["id"] != tt.wantID {
				t.Fatalf("id = %v", dest["id"])
			}
			if _, ok := dest["title"].(string); !ok {
				t.Fatalf("title = %T", dest["title"])
			}
		})
	}

	if err := DecodeJSON([]byte(`[]`), nil); err == nil {
		t.Fatal("nil dest succeeded")
	}

	var raw any
	if err := DecodeJSON([]byte(`[{"id":1}]`), &raw); err != nil {
		t.Fatal(err)
	}
	items, ok := raw.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("array decode = %#v", raw)
	}
	entry, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("entry = %T", items[0])
	}
	if _, ok := entry["id"].(json.Number); !ok {
		t.Fatalf("UseNumber was not applied: %T", entry["id"])
	}
}

type exitError struct{ code int }

func (e exitError) Error() string { return "exit status" }
func (e exitError) ExitCode() int { return e.code }
