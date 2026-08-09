package cli

import (
	"errors"
	"io"
)

type environmentPromptInput struct {
	readLine lineReader
	restore  func() error
}

func terminalLineReader(readLine func() (string, error)) lineReader {
	return func() (string, bool, error) {
		line, err := readLine()
		if errors.Is(err, io.EOF) {
			return "", false, nil
		}
		if err != nil {
			return "", false, err
		}
		return line, true, nil
	}
}
