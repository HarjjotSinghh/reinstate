package main

import (
	"net"
	"os"
	"testing"
	"time"
)

// withRegistryDir points the registry at a temp directory for the
// duration of the test, so these tests never touch this host's real
// per-user cache directory.
func withRegistryDir(t *testing.T) {
	t.Helper()
	t.Setenv("LOCALAPPDATA", t.TempDir()) // os.UserCacheDir() on windows
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
}

func TestRegistryFileIsStablePerRootAndDiffersAcrossRoots(t *testing.T) {
	withRegistryDir(t)
	a, err := registryFile(`D:\lab-a`)
	if err != nil {
		t.Fatal(err)
	}
	again, err := registryFile(`D:\lab-a`)
	if err != nil {
		t.Fatal(err)
	}
	if a != again {
		t.Fatalf("registryFile(root) is not stable: %q vs %q", a, again)
	}
	b, err := registryFile(`D:\lab-b`)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("registryFile gave the same path for two different roots: %q", a)
	}
}

func TestRegisterListUnregisterRoundTrip(t *testing.T) {
	withRegistryDir(t)
	root := t.TempDir()
	e := registryEntry{Root: root, HopdPID: 111, HopdAddr: "127.0.0.1:8082", LockerPID: 222, LockerAddr: "127.0.0.1:9002", StartedAt: time.Now().UTC()}
	if err := registerLab(e); err != nil {
		t.Fatalf("registerLab: %v", err)
	}
	got, err := listRegistry()
	if err != nil {
		t.Fatalf("listRegistry: %v", err)
	}
	if len(got) != 1 || got[0].Root != root || got[0].HopdPID != 111 {
		t.Fatalf("listRegistry = %+v, want one entry for %s", got, root)
	}
	if err := unregisterLab(root); err != nil {
		t.Fatalf("unregisterLab: %v", err)
	}
	got, err = listRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("listRegistry after unregister = %+v, want empty", got)
	}
}

func TestUnregisterLabOnAMissingEntryIsNotAnError(t *testing.T) {
	withRegistryDir(t)
	if err := unregisterLab(t.TempDir()); err != nil {
		t.Fatalf("unregisterLab on a never-registered root: %v", err)
	}
}

func TestRegisterLabReplacesAnEarlierEntryForTheSameRoot(t *testing.T) {
	withRegistryDir(t)
	root := t.TempDir()
	if err := registerLab(registryEntry{Root: root, HopdPID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := registerLab(registryEntry{Root: root, HopdPID: 2}); err != nil {
		t.Fatal(err)
	}
	got, err := listRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].HopdPID != 2 {
		t.Fatalf("listRegistry = %+v, want exactly one entry with HopdPID=2 (the restart replaced the earlier one)", got)
	}
}

func TestProcessAliveFalseForAnImpossiblePID(t *testing.T) {
	// PID 0 is never a real user process on Windows or POSIX.
	if processAlive(0) {
		t.Fatal("processAlive(0) = true, want false")
	}
	if processAlive(-1) {
		t.Fatal("processAlive(-1) = true, want false")
	}
}

func TestProcessAliveTrueForThisProcess(t *testing.T) {
	if !processAlive(os.Getpid()) {
		t.Fatalf("processAlive(os.Getpid()=%d) = false, want true", os.Getpid())
	}
}

func TestRefuseListeningPortCatchesAnAlreadyBoundAddress(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("could not bind a loopback port to test against: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()
	if err := refuseListeningPort(addr, "test"); err == nil {
		t.Fatalf("refuseListeningPort(%s): want an error, the port is already bound", addr)
	}
}

func TestRefuseListeningPortAllowsAFreePort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("could not bind a loopback port to find a free one: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() // free it again; nothing else should grab it in the meantime on a test host
	if err := refuseListeningPort(addr, "test"); err != nil {
		t.Fatalf("refuseListeningPort(%s) on a free port: %v, want nil", addr, err)
	}
}
