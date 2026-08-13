//go:build windows

package agentcheck

import (
	"os/exec"
	"time"
)

func configureVersionCommand(command *exec.Cmd) {
	command.WaitDelay = 200 * time.Millisecond
}
