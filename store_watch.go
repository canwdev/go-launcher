package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// 用户手动编辑 .go-launcher-data/go-launcher-data.json 后自动生效：
// 用文件系统事件监听盯住数据目录（Windows 实现见 store_watch_windows.go，
// 基于 ReadDirectoryChangesW，不轮询），事件到达后重新比对文件指纹
// （mtime/size + 内容 hash）再重载内存 store，并通过 "store:reloaded" 把新数据
// 推给前端。程序自己写入触发的事件靠内容 hash 识别后忽略，不会自我触发。
// 内容写坏（JSON 非法）时保留内存中的旧数据，只发一次 "store:reload-failed"，
// 避免把编辑到一半的文件当空 store 加载。
const (
	// storeWatchQuiet 是事件合并窗口：一次保存往往触发多条事件（写临时文件 +
	// 改名、分块写入等），等写入停下来再读文件，避免读到写了一半的内容。
	storeWatchQuiet = 150 * time.Millisecond
	// storeWatchRetry 是监听建立失败（目录被删、句柄失效等）后的重试间隔。
	storeWatchRetry = 2 * time.Second
)

// errStoreWatchUnsupported 表示当前平台没有实现文件监听（见 store_watch_other.go）。
var errStoreWatchUnsupported = errors.New("store file watching is not supported on this platform")

// storeFileState 是 saveFile 最近一次已知的磁盘状态：mtime/size 用于廉价的
// “没变”判断，内容 hash 用于区分自己的写入与外部编辑。
type storeFileState struct {
	modTime time.Time
	size    int64
	hash    string
}

// matches 报告磁盘上的文件是否与记录的状态一致（mtime + size）。
func (s storeFileState) matches(info os.FileInfo) bool {
	return !s.modTime.IsZero() && s.size == info.Size() && s.modTime.Equal(info.ModTime())
}

func contentHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// storeReloadResult 描述一次 reloadStoreIfChanged 的结果。
type storeReloadResult int

const (
	storeReloadNone    storeReloadResult = iota // 文件没变 / 是自己的写入 / 暂时读不到
	storeReloadApplied                          // 已应用外部改动
	storeReloadInvalid                          // 外部内容不是合法 JSON，已忽略
)

// syncStoreFileState 记录磁盘上当前的 store 内容（启动时调用一次）。
func (a *App) syncStoreFileState() {
	data, err := os.ReadFile(saveFile)
	if err != nil {
		a.storeFile = storeFileState{}
		return
	}
	a.rememberStoreFile(data)
}

// rememberStoreFile 记录刚写入（或刚读入）的文件指纹。调用方必须持有 a.mu，
// 或在构造阶段单线程调用。
func (a *App) rememberStoreFile(data []byte) {
	state := storeFileState{hash: contentHash(data)}
	if info, err := os.Stat(saveFile); err == nil {
		state.modTime = info.ModTime()
		state.size = info.Size()
	}
	a.storeFile = state
}

// reloadStoreIfChanged 检查 saveFile 是否被外部（手动编辑）改动，有则重载内存
// store。调用方不得持有 a.mu；返回本次是否应用了改动。它只改内存，不推送事件，
// 由调用方（监听回调 / GetData）决定是否通知前端。
func (a *App) reloadStoreIfChanged() storeReloadResult {
	info, err := os.Stat(saveFile)
	if err != nil {
		return storeReloadNone // 文件暂不存在（正在被原子替换 / 被删除）：等下一次事件
	}
	a.mu.Lock()
	known := a.storeFile
	a.mu.Unlock()
	if known.matches(info) {
		return storeReloadNone
	}
	data, err := os.ReadFile(saveFile)
	if err != nil {
		return storeReloadNone
	}
	hash := contentHash(data)
	// 读取期间文件可能又被替换，重新取一次指纹作为新的基准。
	info, err = os.Stat(saveFile)
	if err != nil {
		return storeReloadNone
	}
	next := storeFileState{modTime: info.ModTime(), size: info.Size(), hash: hash}

	a.mu.Lock()
	if hash == a.storeFile.hash {
		// 内容与我们最近一次写入一致：只是 mtime/size 变了（例如自己的写入），
		// 不算外部改动。
		a.storeFile = next
		a.mu.Unlock()
		return storeReloadNone
	}
	store, ok := parseStore(data)
	if !ok {
		// 记住这份坏内容，避免同一份坏文件被重复报错。
		a.storeFile = next
		a.mu.Unlock()
		return storeReloadInvalid
	}
	a.applyStoreLocked(store)
	a.storeFile = next
	a.mu.Unlock()

	// 窗口级设置（置顶）也可能被手改，立即重新应用（ctx 为空时内部忽略）。
	applyAlwaysOnTop(a.ctx, store.Settings.AlwaysOnTop)
	return storeReloadApplied
}

// applyStoreLocked 用外部编辑后的 store 替换内存 store：runtime 统计以文件为准
// （用户可能手改了 runtime_ms），被删掉的条目放弃跟踪（进程继续运行，只是不再
// 计时 / 无法 Stop）。调用方必须持有 a.mu。
func (a *App) applyStoreLocked(store AppStore) {
	stats := make(map[string]int64, len(store.Apps))
	for guid, item := range store.Apps {
		if item != nil {
			stats[guid] = item.RuntimeMs
		}
	}
	a.runtimeStats = stats
	for guid := range a.running {
		if store.Apps[guid] == nil {
			delete(a.running, guid)
		}
	}
	a.store = store
	// 这里不做 pruneIconFiles：外部编辑可能只是临时删掉条目，直接删缓存图标不可逆，
	// 交给下一次正常保存（SaveData / 退出）清理。
}

// startStoreWatcher 启动外部改动监听（startup 时调用一次）。
func (a *App) startStoreWatcher() {
	a.watchStart.Do(func() {
		go a.watchStoreLoop()
	})
}

// watchStoreLoop 盯住 store 所在的目录，直到收到停止信号或平台不支持。
func (a *App) watchStoreLoop() {
	dir, err := filepath.Abs(filepath.Dir(saveFile))
	if err != nil {
		return
	}
	name := filepath.Base(saveFile)
	for {
		select {
		case <-a.watchStop:
			return
		default:
		}
		err := watchStoreFile(dir, name, a.watchStop, a.handleStoreFileEvent)
		if err == nil || errors.Is(err, errStoreWatchUnsupported) {
			return // 停止信号 / 平台未实现监听
		}
		// 目录被删、句柄失效等临时错误：稍后重试，别让监听永久失效。
		select {
		case <-a.watchStop:
			return
		case <-time.After(storeWatchRetry):
		}
	}
}

// handleStoreFileEvent 处理一次“数据目录里的 store 文件被动过”的事件。
func (a *App) handleStoreFileEvent() {
	switch a.reloadStoreIfChanged() {
	case storeReloadApplied:
		a.emitStoreReloaded()
	case storeReloadInvalid:
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "store:reload-failed", saveFile)
		}
	}
}

// stopStoreWatcher 结束监听（shutdown 时调用；重复调用安全）。
func (a *App) stopStoreWatcher() {
	a.watchStopOnce.Do(func() { close(a.watchStop) })
}

// emitStoreReloaded 把重载后的完整数据推给前端（前端整体替换本地 store）。
func (a *App) emitStoreReloaded() {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "store:reloaded", a.GetData())
}
