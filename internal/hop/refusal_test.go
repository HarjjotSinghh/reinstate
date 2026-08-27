package hop

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// pollingServer answers each poll with the next function in answers and
// counts the calls, so a test can see a loop stop as well as see its result.
func pollingServer(t *testing.T, answers ...func(w http.ResponseWriter)) (*Client, *int) {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if calls >= len(answers) {
			t.Errorf("poll %d: the client asked again after a terminal answer", calls+1)
			w.WriteHeader(http.StatusInternalServerError)
			calls++
			return
		}
		answer := answers[calls]
		calls++
		answer(w)
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL), &calls
}

func refusedBody(status int, code, reason string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": StatusRefused, "code": code, "reason": reason})
	}
}

func noSleep(context.Context, time.Duration) error { return nil }

// TestARefusalIsReadFromTheBodyNotTheStatus states the rule the protocol
// asks of a client: switch on the code, never on the HTTP status. Every
// refusal answers 403 today, but a client keyed on 403 would read the first
// one answered with anything else as an ordinary error and poll on to a
// timeout — which is the failure the refused answer exists to end.
func TestARefusalIsReadFromTheBodyNotTheStatus(t *testing.T) {
	tests := []struct {
		name   string
		answer func(w http.ResponseWriter)
		// wantRefused is the code a *RefusedError must carry, or "" when
		// the answer is not a refusal at all.
		wantRefused string
		wantReason  string
		wantOther   string
	}{
		{
			name:        "the 403 the control plane sends today",
			answer:      refusedBody(http.StatusForbidden, CodeQuotaDevices, "This account already has the 5 devices the hop plan allows."),
			wantRefused: CodeQuotaDevices,
			wantReason:  "This account already has the 5 devices the hop plan allows.",
		},
		{
			name:        "a refusal carried by some other status",
			answer:      refusedBody(http.StatusTeapot, CodeAccountLinked, "Sign in by email instead."),
			wantRefused: CodeAccountLinked,
			wantReason:  "Sign in by email instead.",
		},
		{
			name:        "a refusal carried by a 200",
			answer:      refusedBody(http.StatusOK, CodeInternalError, "This sign-in did not finish."),
			wantRefused: CodeInternalError,
			wantReason:  "This sign-in did not finish.",
		},
		{
			name:        "a code this build has never heard of",
			answer:      refusedBody(http.StatusForbidden, "account_suspended", "This account is suspended."),
			wantRefused: "account_suspended",
			wantReason:  "This account is suspended.",
		},
		{
			name:        "a refusal with no sentence still has one",
			answer:      refusedBody(http.StatusForbidden, CodeLoginExpired, "  "),
			wantRefused: CodeLoginExpired,
			wantReason:  GenericRefusalReason,
		},
		{
			name: "an ordinary 403 that is not a refusal",
			answer: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			},
			wantOther: "forbidden",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, calls := pollingServer(t, test.answer)
			status, approval, err := client.Poll(t.Context(), LoginSession{ID: "s1", PollSecret: "p"})
			if approval != nil {
				t.Fatalf("a non-approval produced an approval: %+v", approval)
			}
			var refused *RefusedError
			switch {
			case test.wantRefused != "":
				if !errors.As(err, &refused) {
					t.Fatalf("err = %v (%T), want a *RefusedError", err, err)
				}
				if refused.Code != test.wantRefused {
					t.Errorf("code = %q, want %q", refused.Code, test.wantRefused)
				}
				if refused.Sentence() != test.wantReason || refused.Error() != test.wantReason {
					t.Errorf("sentence = %q, want %q", refused.Sentence(), test.wantReason)
				}
				if status != StatusRefused {
					t.Errorf("status = %q, want %q", status, StatusRefused)
				}
			default:
				if errors.As(err, &refused) {
					t.Fatalf("an ordinary error was read as a refusal: %+v", refused)
				}
				if err == nil || err.Error() != test.wantOther {
					t.Fatalf("err = %v, want %q", err, test.wantOther)
				}
			}
			if *calls != 1 {
				t.Errorf("polled %d times for one answer", *calls)
			}
		})
	}
}

// TestWaitForApprovalStopsAtTheFirstRefusal is the loop half of the same
// property. A refusal is terminal: the session is never approved
// afterwards, so asking again can only end in the deadline that used to be
// the whole of this bug.
func TestWaitForApprovalStopsAtTheFirstRefusal(t *testing.T) {
	client, calls := pollingServer(t,
		func(w http.ResponseWriter) { _ = json.NewEncoder(w).Encode(map[string]string{"status": StatusPending}) },
		refusedBody(http.StatusForbidden, CodeQuotaDevices, "This account already has the 5 devices the hop plan allows."),
	)
	approval, err := client.WaitForApproval(t.Context(), LoginSession{ID: "s1", PollSecret: "p"}, noSleep)
	if approval != nil {
		t.Fatalf("approval %+v", approval)
	}
	var refused *RefusedError
	if !errors.As(err, &refused) {
		t.Fatalf("err = %v (%T), want a *RefusedError", err, err)
	}
	if *calls != 2 {
		t.Fatalf("polled %d times; the loop must stop at the refusal", *calls)
	}
}

// TestARefusedAnswerIsNeverAnApproval: a refusal carries no device token,
// account or device. If one ever arrived carrying them, they would be a
// token for a device that was not enrolled.
func TestARefusedAnswerIsNeverAnApproval(t *testing.T) {
	client, _ := pollingServer(t, func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": StatusRefused, "code": CodeQuotaDevices, "reason": "Too many devices.",
			"device_token": "hop_smuggled", "account": Account{ID: "a"}, "device": Device{ID: "d"},
		})
	})
	status, approval, err := client.Poll(t.Context(), LoginSession{ID: "s1", PollSecret: "p"})
	if approval != nil {
		t.Fatalf("a refusal was turned into an approval: %+v", approval)
	}
	if status != StatusRefused {
		t.Fatalf("status = %q", status)
	}
	var refused *RefusedError
	if !errors.As(err, &refused) {
		t.Fatalf("err = %v", err)
	}
}

// TestSignInRefusalCodesAreADistinctCopy: the accessor hands out codes, not
// a handle on the package's own list.
func TestSignInRefusalCodesAreADistinctCopy(t *testing.T) {
	codes := SignInRefusalCodes()
	if len(codes) == 0 {
		t.Fatal("no sign-in refusal codes are declared")
	}
	seen := map[string]bool{}
	for _, code := range codes {
		if code == "" {
			t.Error("an empty refusal code is declared")
		}
		if seen[code] {
			t.Errorf("%q is declared twice", code)
		}
		seen[code] = true
	}
	codes[0] = "mutated"
	if SignInRefusalCodes()[0] == "mutated" {
		t.Fatal("SignInRefusalCodes hands out the package's own slice")
	}
}
