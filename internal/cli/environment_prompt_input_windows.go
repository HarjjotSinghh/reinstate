//go:build windows

package cli

import (
	"bufio"
	"io"

	"golang.org/x/term"
)

type promptReadWriter struct {
	io.Reader
	io.Writer
}

func newEnvironmentPromptInput(input io.Reader, output io.Writer) (environmentPromptInput, error) {
	file, ok := input.(interface{ Fd() uintptr })
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return environmentPromptInput{
			readLine: scannerLineReader(bufio.NewScanner(input)),
			restore:  func() error { return nil },
		}, nil
	}

	state, err := term.MakeRaw(int(file.Fd()))
	if err != nil {
		return environmentPromptInput{}, err
	}
	terminal := term.NewTerminal(promptReadWriter{Reader: input, Writer: output}, "")
	return environmentPromptInput{
		readLine: terminalLineReader(terminal.ReadLine),
		restore:  func() error { return term.Restore(int(file.Fd()), state) },
	}, nil
}
