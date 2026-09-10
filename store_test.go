package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestStorePersistsRunAsAdmin 验证 item 编辑弹窗里的“Run as administrator”勾选：
// 写入 store 文件、写出的 JSON 键名可被前端/后端读回，未勾选的条目保持不写该键。
func TestStorePersistsRunAsAdmin(t *testing.T) {
	app := newTestApp(t)
	app.mu.Lock()
	app.store.Apps["admin"] = &AppItem{GUID: "admin", Name: "Admin", Path: "C:/admin.exe", RunAsAdmin: true}
	app.store.Apps["plain"] = &AppItem{GUID: "plain", Name: "Plain", Path: "C:/plain.exe"}
	app.store.Categories[0].Slots = []*string{ptrString("admin"), ptrString("plain")}
	app.writeStore()
	app.mu.Unlock()

	data, err := os.ReadFile(saveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"run_as_admin": true`) {
		t.Fatalf("run_as_admin must be persisted, got:\n%s", data)
	}

	var raw struct {
		Apps map[string]map[string]any `json:"apps"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw.Apps["plain"]["run_as_admin"]; ok {
		t.Fatalf("items without the flag must omit run_as_admin, got %v", raw.Apps["plain"])
	}

	store, ok := parseStore(data)
	if !ok {
		t.Fatal("parseStore rejected the store written by writeStore")
	}
	if store.Apps["admin"] == nil || !store.Apps["admin"].RunAsAdmin {
		t.Fatalf("run_as_admin did not survive a reload: %+v", store.Apps["admin"])
	}
	if store.Apps["plain"] == nil || store.Apps["plain"].RunAsAdmin {
		t.Fatalf("run_as_admin must default to false: %+v", store.Apps["plain"])
	}

	// 前端保存整份 store 的路径（SaveData）同样要保留该字段
	if err := app.SaveData(store); err != nil {
		t.Fatal(err)
	}
	app.mu.Lock()
	got := app.store.Apps["admin"] != nil && app.store.Apps["admin"].RunAsAdmin
	app.mu.Unlock()
	if !got {
		t.Fatal("SaveData dropped run_as_admin")
	}
}

// TestParseStoreReadsHandEditedRunAsAdmin 覆盖用户手改配置文件写入该字段的场景。
func TestParseStoreReadsHandEditedRunAsAdmin(t *testing.T) {
	store, ok := parseStore([]byte(`{
		"apps": {"a": {"guid": "a", "name": "A", "path": "C:/a.exe", "run_as_admin": true}},
		"categories": [{"guid": "c", "name": "Main", "slots": ["a"]}],
		"settings": {}
	}`))
	if !ok {
		t.Fatal("parseStore rejected valid JSON")
	}
	if store.Apps["a"] == nil || !store.Apps["a"].RunAsAdmin {
		t.Fatalf("hand-edited run_as_admin not read: %+v", store.Apps["a"])
	}
}
