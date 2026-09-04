//go:build !windows

package main

import "errors"

var errShortcutUnsupported = errors.New(".lnk shortcuts are only supported on Windows")

// resolveShortcut is a no-op on non-Windows platforms; .lnk files are added as
// plain files instead.
func resolveShortcut(path string) (shortcutInfo, error) {
	return shortcutInfo{}, errShortcutUnsupported
}
