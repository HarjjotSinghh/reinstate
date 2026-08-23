// Command svccheck exercises the daemon's OS service manager against the
// real scheduler on this host: it renders the definition, installs it,
// reads status, starts and stops it, and removes it, printing each step.
// It registers a throwaway label under a temp command so a bench run can
// prove the generated plist / systemd unit / Task Scheduler XML is
// accepted by the OS without provisioning an account. It always cleans up.
//
//	go run ./scripts/testing/svccheck -exe C:\path\to\rein.exe -label com.reinstate.daemon.svccheck
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/daemon"
)

func main() {
	exe := flag.String("exe", "", "executable the service runs (defaults to this binary)")
	label := flag.String("label", "com.reinstate.daemon.svccheck", "throwaway service label")
	flag.Parse()
	if *exe == "" {
		self, err := os.Executable()
		if err != nil {
			fail("os.Executable", err)
		}
		*exe = self
	}
	userHome, _ := os.UserHomeDir()
	manager, err := daemon.NewManager(runtime.GOOS, userHome, nil)
	if err != nil {
		fail("NewManager", err)
	}
	spec := daemon.Spec{
		Label:      *label,
		Executable: *exe,
		Args:       []string{"version"},
		LogPath:    os.TempDir() + string(os.PathSeparator) + *label + ".log",
		Path:       os.Getenv("PATH"),
	}
	ctx := context.Background()
	fmt.Printf("manager: %s\n", manager.Kind())
	fmt.Printf("definition path: %s\n\n", manager.DefinitionPath(spec))
	rendered, err := manager.Render(spec)
	if err != nil {
		fail("Render", err)
	}
	fmt.Printf("--- rendered definition ---\n%s\n---------------------------\n\n", rendered)

	// Always remove the label first and last so a prior failed run cannot
	// leave a task behind.
	_ = manager.Uninstall(ctx, spec)
	defer func() {
		if err := manager.Uninstall(ctx, spec); err != nil {
			fmt.Printf("cleanup uninstall: %v\n", err)
		} else {
			fmt.Println("cleanup: removed")
		}
	}()

	step("install", manager.Install(ctx, spec))
	printStatus(ctx, manager, spec, "after install")
	step("stop", manager.Stop(ctx, spec))
	printStatus(ctx, manager, spec, "after stop")
	step("start", manager.Start(ctx, spec))
	time.Sleep(time.Second)
	printStatus(ctx, manager, spec, "after start")
	step("uninstall", manager.Uninstall(ctx, spec))
	printStatus(ctx, manager, spec, "after uninstall")
}

func step(name string, err error) {
	if err != nil {
		fmt.Printf("%s: FAILED: %v\n", name, err)
		return
	}
	fmt.Printf("%s: ok\n", name)
}

func printStatus(ctx context.Context, m daemon.Manager, spec daemon.Spec, when string) {
	state, err := m.Status(ctx, spec)
	if err != nil {
		fmt.Printf("status %s: error %v\n", when, err)
		return
	}
	fmt.Printf("status %s: installed=%v running=%v detail=%q\n", when, state.Installed, state.Running, state.Detail)
}

func fail(what string, err error) {
	fmt.Printf("%s: %v\n", what, err)
	os.Exit(1)
}
