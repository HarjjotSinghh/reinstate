//go:build !windows

package cli

import (
	"bufio"
	"io"
)

func newEnvironmentPromptInput(input io.Reader, _ io.Writer) (environmentPromptInput, error) {
	return environmentPromptInput{
		readLine: scannerLineReader(bufio.NewScanner(input)),
		restore:  func() error { return nil },
	}, nil
}
