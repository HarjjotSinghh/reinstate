// Command phase3perf generates and measures the frozen Phase 3 RC1
// performance corpus. It is an acceptance harness, not product runtime code.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "phase3perf:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: phase3perf generate|run [flags]")
	}
	spec, err := loadSpec()
	if err != nil {
		return err
	}
	switch args[0] {
	case "generate":
		flags := flag.NewFlagSet("generate", flag.ContinueOnError)
		root := flags.String("root", "", "new absolute fixture root")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("generate accepts flags only")
		}
		manifest, err := generateFixture(*root, spec)
		if err != nil {
			return err
		}
		return writeCommandJSON(manifest)
	case "run":
		flags := flag.NewFlagSet("run", flag.ContinueOnError)
		config := runConfig{}
		flags.StringVar(&config.Root, "root", "", "new absolute performance evidence root")
		flags.StringVar(&config.Rein, "rein", "", "absolute installed rein path")
		flags.StringVar(&config.Reinstate, "reinstate", "", "absolute installed reinstate path")
		flags.StringVar(&config.SourceRoot, "source-root", "", "absolute exact tagged source checkout")
		flags.StringVar(&config.ExpectedCommit, "expected-commit", "", "literal full tagged commit")
		flags.StringVar(&config.ExpectedVersion, "expected-version", "0.3.0-rc.1", "installed version value")
		flags.StringVar(&config.CuratedPath, "path", "", "curated PATH resolving rein dependencies, Claude, Codex, and Git but not OpenCode")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("run accepts flags only")
		}
		result, err := runHarness(config, spec)
		if err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(config.Root, "results.json"), result); err != nil {
			return err
		}
		return writeCommandJSON(result)
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func writeCommandJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
