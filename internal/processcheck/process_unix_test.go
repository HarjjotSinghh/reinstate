//go:build !windows

package processcheck

import "testing"

func TestParseLsofWorkingDirectories(t *testing.T) {
	dirs := parseLsofWorkingDirectories("p100\nn/home/dev/projects/acceptance\np200\nn/tmp\n")
	if dirs[100] != "/home/dev/projects/acceptance" || dirs[200] != "/tmp" {
		t.Fatalf("unexpected parse: %v", dirs)
	}
	if len(parseLsofWorkingDirectories("")) != 0 {
		t.Fatal("empty lsof output produced entries")
	}
}
