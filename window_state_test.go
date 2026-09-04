package main

import (
	"os"
	"testing"
)

func TestWindowStateRoundTrip(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	if _, err := os.Stat(windowStateFile); !os.IsNotExist(err) {
		t.Fatalf("no window-state file should exist before saving, found one at %q", windowStateFile)
	}

	want := windowState{Width: 900, Height: 600, X: 120, Y: 80, Maximised: true}
	saveWindowState(want)
	got := loadWindowState()
	if got != want {
		t.Fatalf("loadWindowState() = %+v, want %+v", got, want)
	}
}

func TestWindowStateInvalidSizeNotPersisted(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	saveWindowState(windowState{Width: 0, Height: 0, Maximised: true})
	saveWindowState(windowState{Width: 640, Height: -5, Maximised: false})
	if _, err := os.Stat(windowStateFile); !os.IsNotExist(err) {
		t.Fatalf("invalid sizes must not be written, found a state file at %q", windowStateFile)
	}
	if got := loadWindowState(); got != (windowState{}) {
		t.Fatalf("loadWindowState() = %+v, want zero state", got)
	}
}
