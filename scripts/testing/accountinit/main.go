// Command accountinit runs `rein account init` without a terminal: it
// captures the recovery code the command shows and answers the
// confirmation prompt with it, then prints the command's output with the
// code redacted. It exists for bench runs (a Windows scheduled task, an SSH
// session) where no hidden prompt can be answered; it never stores the
// code and never prints it.
//
//	go run ./scripts/testing/accountinit
package main

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/cli"
)

var codePattern = regexp.MustCompile(`\n\s+([A-Z0-9]{4}(?:-[A-Z0-9]{4}){3,})\s*\n`)

func main() {
	var stderr bytes.Buffer
	code := cli.Execute(cli.Options{
		Name: "rein", Args: []string{"account", "init"},
		Stdout: os.Stdout, Stderr: &stderr, Stdin: strings.NewReader(""),
		RecoveryCodePrompt: func(prompt string) ([]byte, error) {
			match := codePattern.FindSubmatch(stderr.Bytes())
			if match == nil {
				return nil, errors.New("recovery code was not shown before the confirmation prompt")
			}
			return match[1], nil
		},
	})
	redacted := codePattern.ReplaceAllString(stderr.String(), "\n    <recovery code redacted>\n")
	_, _ = os.Stderr.WriteString(redacted)
	os.Exit(code)
}
