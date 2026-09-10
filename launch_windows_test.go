//go:build windows

package main

import "testing"

// TestLaunchVerb 固定“以管理员身份启动”的动词选择。这里只验证动词，不实际启动
// 进程——真实提权会在测试机上弹 UAC。
func TestLaunchVerb(t *testing.T) {
	if got := launchVerb(true); got != "runas" {
		t.Fatalf("elevated items must launch with the runas verb, got %q", got)
	}
	if got := launchVerb(false); got != "open" {
		t.Fatalf("normal items must launch with the open verb, got %q", got)
	}
}
