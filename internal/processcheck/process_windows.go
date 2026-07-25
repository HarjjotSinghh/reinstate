//go:build windows

package processcheck

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"os/exec"
)

func agentActive(ctx context.Context, agent string) (bool, error) {
	const command = `Get-CimInstance Win32_Process | Select-Object Name,CommandLine | ConvertTo-Csv -NoTypeInformation`
	output, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command).Output()
	if err == nil {
		reader := csv.NewReader(bytes.NewReader(output))
		if _, err := reader.Read(); err != nil {
			return false, err
		}
		for {
			record, err := reader.Read()
			if err == io.EOF {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			if len(record) >= 2 && matchesAgentProcess(agent, record[0], record[1]) {
				return true, nil
			}
		}
	}

	output, err = exec.CommandContext(ctx, "tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return false, err
	}
	reader := csv.NewReader(bytes.NewReader(output))
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, err
		}
		if len(record) > 0 && matchesAgentProcess(agent, record[0], "") {
			return true, nil
		}
	}
	return false, nil
}
