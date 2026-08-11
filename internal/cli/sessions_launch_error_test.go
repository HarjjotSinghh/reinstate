package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestLocalLaunchErrorNonTTYMessage(t *testing.T) {
	t.Parallel()
	err := localLaunchError(sessionindex.ErrNonInteractiveLaunch)
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error type = %T", err)
	}
	if exitErr.Code != ExitSafety {
		t.Fatalf("code = %d, want %d", exitErr.Code, ExitSafety)
	}
	if !strings.Contains(strings.ToLower(exitErr.Message), "interactive terminal") {
		t.Fatalf("message = %q", exitErr.Message)
	}
}
