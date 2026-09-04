package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// installDir 返回当前可执行文件所在的安装目录（绝对路径，尽量归一化）。
func installDir() string {
	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}
	dir := filepath.Dir(exe)
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = filepath.Clean(dir)
	}
	return filepath.Clean(abs)
}

// installKey 返回当前安装目录对应的单例键。
//
// 单例按安装目录划分：每个安装目录只放一份 go-launcher.exe，目录不同则键不同，
// 不同目录的程序可以同时运行；同一目录再次启动会得到相同键，被判定为重复实例，
// 由调用方聚焦已有窗口后退出。
//
// Windows 大小写折叠保证同一目录永远得到同一个键；哈希输出定长十六进制，可安全
// 用于命名互斥体 / 锁文件名。
func installKey() string {
	dir := installDir()
	if runtime.GOOS == "windows" {
		dir = strings.ToLower(dir) // Windows 路径大小写不敏感
	}
	sum := sha256.Sum256([]byte(dir))
	return hex.EncodeToString(sum[:])
}

// instanceTitle 返回当前实例的窗口标题：应用名 + 完整安装目录。
//
// 用完整安装目录作为定位标识：FindWindowW 只能按窗口标题精确匹配（大小写不敏感），
// 把完整目录放进标题后，FindWindowW 就能按目录找到对应窗口；且完整目录天然唯一
// （不同安装目录全路径必不同），无需再叠加校验位。标题本身也直接展示安装位置。
func instanceTitle(dir string) string {
	return "Go Launcher - " + dir
}
