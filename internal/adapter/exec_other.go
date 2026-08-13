//go:build !unix && !windows

package adapter

import (
	"os/exec"
	"time"
)

func configureVersionCommand(command *exec.Cmd) {
	command.WaitDelay = 200 * time.Millisecond
}
