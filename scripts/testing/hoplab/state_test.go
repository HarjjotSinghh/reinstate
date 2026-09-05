package main

import (
	"os"
	"testing"
)

func TestLabStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := LabState{
		Root: dir, HopdPID: 1234, HopdAddr: "127.0.0.1:8082", HopdBaseURL: "http://127.0.0.1:8082",
		HopdLog: dir + "/hopd.log", HopdDB: dir + "/hopd.db", LockerPID: 5678, LockerAddr: "127.0.0.1:9002",
	}
	if err := s.save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := loadState(dir)
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if got != s {
		t.Fatalf("loadState = %+v, want %+v", got, s)
	}
}

func TestLoadStateMissing(t *testing.T) {
	dir := t.TempDir()
	if _, err := loadState(dir); !os.IsNotExist(err) {
		t.Fatalf("loadState on an empty dir: err=%v, want IsNotExist", err)
	}
}

func TestRemoveStateIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := removeState(dir); err != nil {
		t.Fatalf("removeState on an already-clean dir: %v", err)
	}
	s := LabState{Root: dir}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	if err := removeState(dir); err != nil {
		t.Fatalf("removeState: %v", err)
	}
	if _, err := loadState(dir); !os.IsNotExist(err) {
		t.Fatalf("state file survived removeState: err=%v", err)
	}
}
