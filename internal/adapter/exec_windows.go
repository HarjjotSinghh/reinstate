//go:build windows

package adapter

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func configureVersionCommand(command *exec.Cmd) {
	command.WaitDelay = 200 * time.Millisecond
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		killWindowsProcessTree(command.Process.Pid)
		return command.Process.Kill()
	}
}

func killWindowsProcessTree(pid int) {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = os.Getenv("SYSTEMROOT")
	}
	taskkill := "taskkill"
	if root != "" {
		taskkill = filepath.Join(root, "System32", "taskkill.exe")
	}
	kill := exec.Command(taskkill, "/T", "/F", "/PID", strconv.Itoa(pid))
	kill.Stdout = io.Discard
	kill.Stderr = io.Discard
	_ = kill.Run()
}
