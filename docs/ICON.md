# Go Launcher 应用图标

本文档说明 Go Launcher 应用图标的文件、设计、使用方式与仓库提交约定。

## 图标文件

| 路径 | 说明 |
| --- | --- |
| `build/appicon.svg` | 图标矢量源（512×512）。修改设计改这个文件。 |
| `build/appicon.png` | Wails 源图标（512×512 透明 PNG），由 `appicon.svg` 生成。 |
| `build/windows/icon.ico` | Windows exe 图标（16/24/32/48/64/128/256 多尺寸透明）。 |
| `build/windows/info.json` | Windows 版本信息模板（打包用）。 |
| `build/windows/wails.exe.manifest` | Windows 应用清单（打包用）。 |
| `docs/generate-icon.ps1` | 图标一键生成脚本。 |


## 使用方式

在项目根目录执行：

```powershell
# 只重新生成图标资源（appicon.png + icon.ico）
.\docs\generate-icon.ps1

# 重新生成图标资源并重建 exe（wails build -clean + 复制 exe 到根目录）
.\docs\generate-icon.ps1 -Build
```

脚本自动完成：SVG 渲染为透明 PNG → 生成多尺寸透明 ICO → 校验透明性 → （可选）重建并复制 exe。

## 注意事项

- 脚本依赖本机 Chrome/Edge 与 .NET System.Drawing（Windows 自带）。
- `generate-icon.ps1` 必须是 **UTF-8 with BOM** 编码（PowerShell 5.1 按 ANSI 解析无 BOM 文件会读乱中文注释）。
- 换了图标后若桌面/任务栏仍显示旧图标，刷新 Windows 图标缓存后重启资源管理器即可（见下）。

## 更换图标后刷新缓存

应用运行时内存图标缓存重启即清空；磁盘缓存 `go-launcher-data/icons/` 不入库。若桌面/任务栏仍显示旧图标，运行以下脚本刷新 Windows 图标缓存：

```powershell
Remove-Item "$env:LOCALAPPDATA\IconCache.db" -Force -ErrorAction SilentlyContinue
Remove-Item "$env:LOCALAPPDATA\Microsoft\Windows\Explorer\iconcache_*.db" -Force -ErrorAction SilentlyContinue
Stop-Process -Name explorer -Force
Start-Process explorer
```

## 仓库提交约定

- **提交**：`build/appicon.svg`、`build/appicon.png`、`build/windows/`（`icon.ico`、`info.json`、`wails.exe.manifest`）、`docs/generate-icon.ps1`、`docs/ICON.md`。
- **不提交**：`build/bin/`（编译产物）、`go-launcher-data/`（运行时数据）——见 `.gitignore`。
