package project

import "testing"

func TestFromGitRemote(t *testing.T) {
	cases := map[string]string{
		"https://github.com/Example/Repo.git": "github.com/Example/Repo",
		"git@github.com:Example/Repo.git":     "github.com/Example/Repo",
	}
	for in, want := range cases {
		got := FromGitRemote(in)
		if got != want {
			t.Fatalf("%s => %q want %q", in, got, want)
		}
	}
}

func TestOpaqueIDStable(t *testing.T) {
	a := OpaqueID("/tmp/foo")
	b := OpaqueID("/tmp/foo")
	if a != b || a == "" {
		t.Fatalf("%s %s", a, b)
	}
}
