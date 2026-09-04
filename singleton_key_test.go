package main

import (
	"encoding/hex"
	"regexp"
	"testing"
)

func TestInstallKeyStableAndHex(t *testing.T) {
	key1 := installKey()
	key2 := installKey()
	if key1 != key2 {
		t.Fatalf("installKey() must be stable within a process: %q vs %q", key1, key2)
	}
	if len(key1) != 64 {
		t.Fatalf("installKey() must return a 64-char SHA-256 hex, got %d chars", len(key1))
	}
	if _, err := hex.DecodeString(key1); err != nil {
		t.Fatalf("installKey() must be valid hex: %v", err)
	}
}

func TestInstanceTitleUsesFullDir(t *testing.T) {
	got := instanceTitle(`C:\games\minecraft`)
	want := "Go Launcher - C:\\games\\minecraft"
	if got != want {
		t.Fatalf("instanceTitle() = %q, want %q", got, want)
	}
	if instanceTitle(`C:\games\minecraft`) == instanceTitle(`D:\tools\photoshop`) {
		t.Fatal("instanceTitle() must differ for different directories")
	}
}

func TestInstanceTitleMatchesTitlePattern(t *testing.T) {
	re := regexp.MustCompile(`^Go Launcher - .+$`)
	if !re.MatchString(instanceTitle(installDir())) {
		t.Fatalf("instanceTitle(installDir()) must match the window-title pattern, got %q", instanceTitle(installDir()))
	}
}
