package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// requireStoreWatch 跳过依赖文件系统监听的用例：监听目前只在 Windows 上实现
// （见 store_watch_windows.go / store_watch_other.go）。
func requireStoreWatch(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("store file watching is only implemented on Windows")
	}
}

// newTestApp 构造一个不依赖 Wails/NewApp 的 App（NewApp 会在仓库目录里建
// .go-launcher-data），并把工作目录切到临时目录，让 saveFile 落在隔离的位置。
func newTestApp(t *testing.T) *App {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	app := &App{
		store:        loadStore(),
		runtimeStats: map[string]int64{},
		running:      map[string]*runningProc{},
		iconCache:    map[string]string{},
		watchStop:    make(chan struct{}),
	}
	app.syncStoreFileState()
	return app
}

// writeManualStore 模拟用户手动编辑 go-launcher-data.json。
func writeManualStore(t *testing.T, store AppStore) {
	t.Helper()
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeManualBytes(t, data)
}

func writeManualBytes(t *testing.T, data []byte) {
	t.Helper()
	if err := os.WriteFile(saveFile, data, 0644); err != nil {
		t.Fatal(err)
	}
	// 保证 mtime 与上一次已知状态不同（部分文件系统时间戳粒度较粗）。
	now := time.Now()
	if err := os.Chtimes(saveFile, now, now); err != nil {
		t.Fatal(err)
	}
}

func testStoreWithHandItem() AppStore {
	return AppStore{
		Apps: map[string]*AppItem{
			"hand": {GUID: "hand", Name: "Hand edited", Path: "C:/tools/hand.exe", RuntimeMs: 12345},
		},
		Categories: []CategoryNode{
			{GUID: "cat-1", Name: "Manual", Slots: []*string{ptrString("hand")}},
		},
	}
}

func ptrString(s string) *string { return &s }

func TestParseStore(t *testing.T) {
	if _, ok := parseStore([]byte("{ not json")); ok {
		t.Fatal("parseStore must reject invalid JSON")
	}

	// 旧文件缺 game_mode / auto_hide：默认打开
	store, ok := parseStore([]byte(`{"apps":{"a":{"guid":"a","path":"C:/a.exe"}},"categories":[{"guid":"c","name":"Main"}]}`))
	if !ok {
		t.Fatal("parseStore rejected valid JSON")
	}
	if !store.Settings.GameMode || !store.Settings.AutoHide {
		t.Fatalf("legacy file should default game_mode/auto_hide to true, got %+v", store.Settings)
	}
	if store.Categories[0].Slots == nil {
		t.Fatal("nil slots must be normalized to an empty slice")
	}

	// 空 store 回退默认（与 loadStore 的既有语义一致）
	empty, ok := parseStore([]byte(`{"apps":{},"categories":[]}`))
	if !ok {
		t.Fatal("parseStore rejected an empty store")
	}
	if len(empty.Categories) != 1 || len(empty.Apps) != 0 {
		t.Fatalf("empty store should fall back to the default store, got %+v", empty)
	}
}

func TestSyncStoreFileStateBaselinesExistingFile(t *testing.T) {
	app := newTestApp(t)
	writeManualStore(t, testStoreWithHandItem())
	app.syncStoreFileState()

	if got := app.reloadStoreIfChanged(); got != storeReloadNone {
		t.Fatalf("a file already fingerprinted at startup must not be reloaded, got %v", got)
	}
}

func TestReloadStoreIfChangedIgnoresOwnWrites(t *testing.T) {
	app := newTestApp(t)

	app.mu.Lock()
	app.store.Apps["own"] = &AppItem{GUID: "own", Name: "Own", Path: "C:/own.exe"}
	app.runtimeStats["own"] = 555
	app.store.Categories[0].Slots = []*string{ptrString("own")}
	app.writeStore()
	app.mu.Unlock()

	if got := app.reloadStoreIfChanged(); got != storeReloadNone {
		t.Fatalf("own write must not be treated as an external edit, got %v", got)
	}
	if app.store.Apps["own"] == nil || app.runtimeStats["own"] != 555 {
		t.Fatalf("own write must keep the in-memory store intact, got %+v", app.store)
	}
}

func TestReloadStoreIfChangedAppliesManualEdit(t *testing.T) {
	app := newTestApp(t)
	app.mu.Lock()
	app.writeStore()
	app.mu.Unlock()

	writeManualStore(t, testStoreWithHandItem())

	if got := app.reloadStoreIfChanged(); got != storeReloadApplied {
		t.Fatalf("manual edit should be applied, got %v", got)
	}
	item := app.store.Apps["hand"]
	if item == nil || item.Name != "Hand edited" {
		t.Fatalf("manual edit not applied: %+v", app.store.Apps)
	}
	if app.runtimeStats["hand"] != 12345 {
		t.Fatalf("hand-edited runtime_ms must win, got %d", app.runtimeStats["hand"])
	}
	if len(app.store.Categories) != 1 || app.store.Categories[0].Name != "Manual" {
		t.Fatalf("manual categories not applied: %+v", app.store.Categories)
	}
	if got := app.reloadStoreIfChanged(); got != storeReloadNone {
		t.Fatalf("the same revision must only be applied once, got %v", got)
	}
}

func TestReloadStoreIfChangedIgnoresInvalidContent(t *testing.T) {
	app := newTestApp(t)
	writeManualStore(t, testStoreWithHandItem())
	if got := app.reloadStoreIfChanged(); got != storeReloadApplied {
		t.Fatalf("manual edit should be applied, got %v", got)
	}

	writeManualBytes(t, []byte(`{"apps": {`))
	if got := app.reloadStoreIfChanged(); got != storeReloadInvalid {
		t.Fatalf("invalid JSON should be reported as invalid, got %v", got)
	}
	if app.store.Apps["hand"] == nil {
		t.Fatal("a broken hand edit must not wipe the in-memory store")
	}
	if got := app.reloadStoreIfChanged(); got != storeReloadNone {
		t.Fatalf("the same broken revision must only be reported once, got %v", got)
	}
}

func TestReloadStoreDropsItemsRemovedOnDisk(t *testing.T) {
	app := newTestApp(t)
	app.mu.Lock()
	app.store.Apps["hand"] = &AppItem{GUID: "hand", Name: "Hand", Path: "C:/hand.exe"}
	app.runtimeStats["hand"] = 999
	app.running["hand"] = &runningProc{}
	app.writeStore()
	app.mu.Unlock()

	writeManualStore(t, AppStore{
		Apps:       map[string]*AppItem{},
		Categories: []CategoryNode{{GUID: "cat-2", Name: "Kept", Slots: []*string{}}},
	})

	if got := app.reloadStoreIfChanged(); got != storeReloadApplied {
		t.Fatalf("manual edit should be applied, got %v", got)
	}
	if _, ok := app.running["hand"]; ok {
		t.Fatal("an item removed on disk must no longer be tracked as running")
	}
	if _, ok := app.runtimeStats["hand"]; ok {
		t.Fatal("an item removed on disk must not keep runtime stats")
	}
	if len(app.store.Categories) != 1 || app.store.Categories[0].Name != "Kept" {
		t.Fatalf("manual categories not applied: %+v", app.store.Categories)
	}
}

func TestGetDataPicksUpManualEditImmediately(t *testing.T) {
	app := newTestApp(t)
	app.mu.Lock()
	app.writeStore()
	app.mu.Unlock()

	writeManualStore(t, testStoreWithHandItem())

	data := app.GetData()
	if data.Store.Apps["hand"] == nil {
		t.Fatal("GetData must re-read a hand-edited file (Refresh menu)")
	}
	if data.State["hand"].RuntimeMs != 12345 {
		t.Fatalf("GetData state should reflect the hand-edited runtime, got %+v", data.State["hand"])
	}
}

func TestStoreWatcherPicksUpManualEdit(t *testing.T) {
	requireStoreWatch(t)
	app := newTestApp(t)
	app.mu.Lock()
	app.writeStore()
	app.mu.Unlock()
	app.startStoreWatcher()
	defer app.stopStoreWatcher()

	writeManualStore(t, testStoreWithHandItem())

	waitForApp(t, app, "hand")
}

// TestStoreWatcherPicksUpReplaceByRename 覆盖编辑器常见的保存方式：写临时文件再
// 原子改名覆盖（此时是目录事件，而不是对目标文件的直接写入）。
func TestStoreWatcherPicksUpReplaceByRename(t *testing.T) {
	requireStoreWatch(t)
	app := newTestApp(t)
	app.mu.Lock()
	app.writeStore()
	app.mu.Unlock()
	app.startStoreWatcher()
	defer app.stopStoreWatcher()

	data, err := json.MarshalIndent(testStoreWithHandItem(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	tmp := saveFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, saveFile); err != nil {
		t.Fatal(err)
	}

	waitForApp(t, app, "hand")
}

// TestStoreWatcherSurvivesDeleteAndRecreate：文件被删掉时保留内存数据，重新出现
// 后再被读入。
func TestStoreWatcherSurvivesDeleteAndRecreate(t *testing.T) {
	requireStoreWatch(t)
	app := newTestApp(t)
	app.mu.Lock()
	app.store.Apps["keep"] = &AppItem{GUID: "keep", Name: "Keep", Path: "C:/keep.exe"}
	app.writeStore()
	app.mu.Unlock()
	app.startStoreWatcher()
	defer app.stopStoreWatcher()

	if err := os.Remove(saveFile); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	app.mu.Lock()
	_, kept := app.store.Apps["keep"]
	app.mu.Unlock()
	if !kept {
		t.Fatal("deleting the file must not wipe the in-memory store")
	}

	writeManualStore(t, testStoreWithHandItem())
	waitForApp(t, app, "hand")
}

// waitForApp 等某个 guid 出现在内存 store 里（监听是异步的）。
func waitForApp(t *testing.T, app *App, guid string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		app.mu.Lock()
		_, found := app.store.Apps[guid]
		app.mu.Unlock()
		if found {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("the file watcher did not pick up %q within 5s", guid)
}

// TestStoreWatcherIgnoresOwnWrites 验证程序自己的写入（writeStore）不会触发重载：
// 落盘后再改内存，如果监听把自己的写入当成外部编辑，内存里的改动会被磁盘内容覆盖。
func TestStoreWatcherIgnoresOwnWrites(t *testing.T) {
	requireStoreWatch(t)
	app := newTestApp(t)
	app.startStoreWatcher()
	defer app.stopStoreWatcher()

	app.mu.Lock()
	app.store.Apps["own"] = &AppItem{GUID: "own", Name: "OnDisk", Path: "C:/own.exe"}
	app.store.Categories[0].Slots = []*string{ptrString("own")}
	app.writeStore()
	app.store.Apps["own"].Name = "InMemoryOnly"
	app.mu.Unlock()

	time.Sleep(time.Second)

	app.mu.Lock()
	got := ""
	if item := app.store.Apps["own"]; item != nil {
		got = item.Name
	}
	app.mu.Unlock()
	if got != "InMemoryOnly" {
		t.Fatalf("own write must not be re-read from disk, got %q", got)
	}
}

func TestStopStoreWatcherIsIdempotent(t *testing.T) {
	app := newTestApp(t)
	app.startStoreWatcher()
	app.stopStoreWatcher()
	app.stopStoreWatcher()
}

// TestWatchStoreFileStartStop 反复建立/停止底层监听，覆盖 stop 时取消挂起 I/O 的
// 路径（卡住、句柄提前关闭等）。
func TestWatchStoreFileStartStop(t *testing.T) {
	requireStoreWatch(t)
	newTestApp(t)
	dir, err := filepath.Abs(filepath.Dir(saveFile))
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(saveFile)
	for i := 0; i < 20; i++ {
		stop := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = watchStoreFile(dir, name, stop, func() {})
		}()
		time.Sleep(5 * time.Millisecond)
		close(stop)
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("watchStoreFile did not stop on iteration %d", i)
		}
	}
}
