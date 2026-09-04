package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func sanitizeBase(name string) string {
	base := strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return '_'
		}
		return r
	}, name)
	if len(base) > 40 {
		base = base[:40]
	}
	return base
}

func normalizePath(path string) string {
	if runtime.GOOS != "windows" {
		return path
	}
	p := strings.ReplaceAll(path, "/", `\`)
	if len(p) >= 3 && p[0] == '\\' && p[2] == ':' &&
		((p[1] >= 'A' && p[1] <= 'Z') || (p[1] >= 'a' && p[1] <= 'z')) {
		p = p[1:]
	}
	return p
}

// shortcutInfo holds the parsed contents of a Windows .lnk shortcut.
type shortcutInfo struct {
	Target  string // resolved program/file path
	Args    string // launch arguments as stored in the shortcut
	WorkDir string // startup ("start in") folder
}

func isShortcutFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".lnk")
}

func defaultTitle(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func formatRuntime(ms int64) string {
	total := ms / 1000
	d := total / 86400
	h := (total % 86400) / 3600
	m := (total % 3600) / 60
	parts := make([]string, 0, 3)
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
	}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if len(parts) == 0 {
		return "1m"
	}
	return strings.Join(parts, " ")
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))
		return ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".com"
	}
	return info.Mode()&0111 != 0
}

func splitArgs(s string) []string {
	var args []string
	var cur strings.Builder
	inQuote := false
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case (r == ' ' || r == '\t') && !inQuote:
			if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		args = append(args, cur.String())
	}
	return args
}
