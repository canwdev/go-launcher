### 最终 TypeScript 类型定义

```typescript
// ============ 应用实体（全局池） ============
interface AppItem {
  guid: string;
  name: string;
  path: string;
  icon?: string;
  runtime_ms?: number;      // 由 Go 后端维护
  args?: string;
  working_dir?: string;
}

// ============ 网格槽位（存应用 guid，null 表示空格） ============
type GridSlot = string | null;

// ============ 分类节点（扁平，无层级，顺序由数组决定） ============
interface CategoryNode {
  guid: string;
  name: string;
  slots: GridSlot[];        // 一维数组，长度由前端维护
}

// ============ 根数据结构 ============
interface AppStore {
  apps: Record<string, AppItem>;   // 全局应用池
  categories: CategoryNode[];      // 分类列表，顺序即显示顺序
  settings: {
    auto_minimize: boolean;
    // 其他全局设置...
  };
}
```

---

### 对应的 JSON 存储示例

```json
{
  "apps": {
    "app-001": {
      "guid": "app-001",
      "name": "CS:GO",
      "path": "steam://rungameid/730",
      "icon": "./icons/csgo.ico",
      "runtime_ms": 1234567,
      "args": "-high",
      "working_dir": "C:\\Steam"
    },
    "app-002": {
      "guid": "app-002",
      "name": "Dism++x64",
      "path": "..\\..\\Programs\\Dism++\\Dism++x64.exe",
      "icon": "./icons/dism.ico",
      "runtime_ms": 39897
    },
    "app-003": {
      "guid": "app-003",
      "name": "KeePassXC",
      "path": "..\\..\\Programs\\KeePassXC\\KeePassXC.exe",
      "runtime_ms": 184401
    }
  },
  "categories": [
    {
      "guid": "cat-game-001",
      "name": "游戏",
      "slots": ["app-001", null, "app-002", "app-003"]
    },
    {
      "guid": "cat-tool-001",
      "name": "系统工具",
      "slots": ["app-002", "app-003", null, null]
    }
  ],
  "settings": {
    "auto_minimize": true
  }
}
```

### 几点补充说明

- 无版本号，不涉及版本更新，直接使用最新数据结构，无需转换。
- 无 `active` 状态，前端在 localStorage 自行维护 active 状态
- categories 就是 tabs 但更灵活
- **应用复用**：同一个 `guid` 可出现在多个分类的 `slots` 中，实现多对多归属，数据仅存一份。当前阶段无需实现前端管理UI。
