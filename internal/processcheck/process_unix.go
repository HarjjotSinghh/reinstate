//go:build !windows

package processcheck

import (
	"context"
	"os/exec"
	"strings"
)

func agentActive(ctx context.Context, agent string) (bool, error) {
	output, err := exec.CommandContext(ctx, "ps", "-axo", "comm=,args=").Output()
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if matchesAgentProcess(agent, fields[0], strings.Join(fields[1:], " ")) {
			return true, nil
		}
	}
	return false, nil
}
