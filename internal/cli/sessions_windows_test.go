package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNativeClaudeLaunchThroughWindowsCommandShim proves the native Windows
// boundary that cannot be exercised on Unix: when exec.LookPath resolves the
// vendor executable to claude.cmd, stdin, stdout, argv, and cwd survive the
// cmd.exe hop used by the real CLI launch path.
//
// The test deliberately uses Execute with the production ExecLaunchRunner
// instead of an injected fake runner. This keeps the result independent of an
// installed or authenticated Claude Code CLI while covering both native
// operations that Device B must drive with piped stdin.
func TestNativeClaudeLaunchThroughWindowsCommandShim(t *testing.T) {
	binDir := t.TempDir()
	workspace := t.TempDir()
	shimPath := filepath.Join(binDir, "claude.cmd")
	shim := strings.Join([]string{
		"@echo off",
		"setlocal",
		`set /p "rein_input="`,
		`if not "%rein_input%"=="%REIN_TEST_STDIN%" exit /b 31`,
		`if not "%~1"=="%REIN_TEST_ARG1%" exit /b 32`,
		`if not "%~2"=="%REIN_TEST_ARG2%" exit /b 33`,
		`if not "%~3"=="%REIN_TEST_ARG3%" exit /b 34`,
		`if /I not "%CD%"=="%REIN_TEST_CWD%" exit /b 35`,
		`echo REINSTATE-WINDOWS-STDIN-INHERITANCE-OK`,
		"exit /b 0",
		"",
	}, "\r\n")
	if err := os.WriteFile(shimPath, []byte(shim), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
	t.Setenv("REIN_TEST_STDIN", "REINSTATE-WINDOWS-CONTROLLED-INPUT")
	t.Setenv("REIN_TEST_ARG1", "--resume")
	t.Setenv("REIN_TEST_ARG2", "claude-one")
	t.Setenv("REIN_TEST_CWD", workspace)

	tests := []struct {
		name      string
		command   string
		thirdArg  string
		wantToken string
	}{
		{
			name:      "resume",
			command:   "resume",
			wantToken: "REINSTATE-WINDOWS-STDIN-INHERITANCE-OK",
		},
		{
			name:      "fork",
			command:   "fork",
			thirdArg:  "--fork-session",
			wantToken: "REINSTATE-WINDOWS-STDIN-INHERITANCE-OK",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("REIN_TEST_ARG3", test.thirdArg)
			stdout, stderr, code := runLocalCLI(
				t,
				localTestSources(workspace)[:1],
				nil,
				"REINSTATE-WINDOWS-CONTROLLED-INPUT\n",
				false,
				test.command,
				"claude:claude-one",
			)
			if code != ExitOK {
				t.Fatalf(
					"%s exit=%d stdout=%q stderr=%q",
					test.command,
					code,
					stdout,
					stderr,
				)
			}
			if !strings.Contains(stdout, test.wantToken) {
				t.Fatalf("%s did not inherit child stdout: %q", test.command, stdout)
			}
			if stderr != "" {
				t.Fatalf("%s wrote unexpected stderr: %q", test.command, stderr)
			}
		})
	}
}
