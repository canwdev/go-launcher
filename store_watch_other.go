//go:build !windows

package main

// watchStoreFile 在非 Windows 平台没有实现文件系统监听（本启动器是 Windows 桌面
// 应用，_other 文件只需保证 go build ./... 通过）。此时手动编辑仍然会在下一次
// GetData 读取时被 reloadStoreIfChanged 的指纹比对拿到（例如菜单里的 Refresh）。
func watchStoreFile(string, string, <-chan struct{}, func()) error {
	return errStoreWatchUnsupported
}
