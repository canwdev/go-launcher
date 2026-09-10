//go:build windows

package main

import (
	"encoding/binary"
	"testing"

	"golang.org/x/sys/windows"
)

type notifyEntry struct {
	action uint32
	name   string
}

// buildNotifyBuffer 拼出一段 ReadDirectoryChangesW 输出格式的缓冲：
// 每条记录 = FILE_NOTIFY_INFORMATION 头（12 字节）+ UTF-16 文件名字节，记录按
// DWORD 对齐，最后一条的 NextEntryOffset 为 0。
func buildNotifyBuffer(entries ...notifyEntry) []byte {
	buf := make([]byte, 0, 64)
	for i, e := range entries {
		name := windows.StringToUTF16(e.name)
		nameBytes := (len(name) - 1) * 2 // 不含结尾 NUL
		rec := make([]byte, 12+nameBytes)
		binary.LittleEndian.PutUint32(rec[4:], e.action)
		binary.LittleEndian.PutUint32(rec[8:], uint32(nameBytes))
		for j := 0; j < len(name)-1; j++ {
			binary.LittleEndian.PutUint16(rec[12+j*2:], name[j])
		}
		if i < len(entries)-1 {
			next := len(rec)
			if pad := next % 4; pad != 0 {
				next += 4 - pad
			}
			binary.LittleEndian.PutUint32(rec[0:], uint32(next))
			rec = append(rec, make([]byte, next-len(rec))...)
		}
		buf = append(buf, rec...)
	}
	return buf
}

func TestStoreFileTouched(t *testing.T) {
	const name = "go-launcher-data.json"

	if storeFileTouched(nil, name) {
		t.Fatal("an empty buffer must not be reported as a hit")
	}
	// window-state.json 的写入不该触发 store 重载
	other := buildNotifyBuffer(notifyEntry{windows.FILE_ACTION_MODIFIED, "window-state.json"})
	if storeFileTouched(other, name) {
		t.Fatal("unrelated file names must be ignored")
	}
	// 一条链里既有无关文件也有目标文件
	mixed := buildNotifyBuffer(
		notifyEntry{windows.FILE_ACTION_MODIFIED, "window-state.json"},
		notifyEntry{windows.FILE_ACTION_MODIFIED, name},
	)
	if !storeFileTouched(mixed, name) {
		t.Fatal("the target file further down the entry chain must be detected")
	}
	// 编辑器常见的“写临时文件再改名”：以 RENAMED_NEW_NAME 出现
	renamed := buildNotifyBuffer(notifyEntry{windows.FILE_ACTION_RENAMED_NEW_NAME, name})
	if !storeFileTouched(renamed, name) {
		t.Fatal("a rename onto the target name must be detected")
	}
	// 大小写不敏感
	upper := buildNotifyBuffer(notifyEntry{windows.FILE_ACTION_ADDED, "GO-LAUNCHER-DATA.JSON"})
	if !storeFileTouched(upper, name) {
		t.Fatal("file name matching must be case-insensitive")
	}
}
