//go:build unix

package adapter

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

func configureVersionCommand(command *exec.Cmd) {
	command.WaitDelay = 200 * time.Millisecond
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
}
