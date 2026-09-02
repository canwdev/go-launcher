# Go Launcher 应用图标

本文档说明 Go Launcher 应用图标的文件、生成方式、注意事项与仓库提交约定。

## 图标文件

| 路径 | 说明 |
| --- | --- |
| `build/appicon.png` | Wails 源图标（512×512 PNG，透明背景）。`wails build` 打包时据此生成各平台图标。 |
| `build/windows/icon.ico` | Windows exe 图标。多尺寸（16/24/32/48/64/128/256），32bpp 透明 PNG 条目。 |
| `build/windows/info.json` | Windows 版本信息模板（打包用）。 |
| `build/windows/wails.exe.manifest` | Windows 应用清单（打包用）。 |

> 换图标前先备份原文件；Wails 默认图标（白色 “W”）可由 `wails build` 重新生成，无需特意保留。

## 源图要求

- 建议 **512×512（或更大）正方形 PNG**，**透明背景**。
- 若源图是白底/纯色背景，需先转为透明背景再执行下述流程（透明背景是应用图标的硬性要求）。

## 生成方式（Windows / PowerShell）

脚本依赖 .NET 的 `System.Drawing`（Windows 自带，无需额外安装）。

### 1) 源图 → `build/appicon.png`

源图本身为透明背景时直接复制即可：

```powershell
Copy-Item "<源图.png>" build\appicon.png
```

### 2) 生成多尺寸透明 `build/windows/icon.ico`

```powershell
Add-Type -AssemblyName System.Drawing
$src = [System.Drawing.Bitmap]::FromFile((Resolve-Path 'build\appicon.png'))
$sizes = @(16, 24, 32, 48, 64, 128, 256)
$pngs = New-Object System.Collections.ArrayList
foreach ($s in $sizes) {
  $bmp = New-Object System.Drawing.Bitmap($s, $s, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
  $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
  $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
  $g.CompositingMode = [System.Drawing.Drawing2D.CompositingMode]::SourceCopy   # 保留 alpha，不要 Clear(White)
  $g.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality
  $g.DrawImage($src, 0, 0, $s, $s)
  $g.Dispose()
  $ms = New-Object System.IO.MemoryStream
  $bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
  [void]$pngs.Add(@($s, $ms.ToArray()))
  $bmp.Dispose(); $ms.Dispose()
}
$src.Dispose()

# 打包 ICO：ICONDIR 头 + 每尺寸 16 字节条目 + PNG 数据
$count = $pngs.Count
$offset = 6 + $count * 16
$ms = New-Object System.IO.MemoryStream
$bw = New-Object System.IO.BinaryWriter($ms)
$bw.Write([uint16]0)    # reserved
$bw.Write([uint16]1)    # type: icon
$bw.Write([uint16]$count)
foreach ($p in $pngs) {
  $s = $p[0]; $data = $p[1]
  $wh = if ($s -ge 256) { 0 } else { $s }
  $bw.Write([byte]$wh); $bw.Write([byte]$wh)
  $bw.Write([byte]0)     # color count
  $bw.Write([byte]0)     # reserved
  $bw.Write([uint16]1)   # planes
  $bw.Write([uint16]32)  # bit count
  $bw.Write([uint32]$data.Length)
  $bw.Write([uint32]$offset)
  $offset += $data.Length
}
foreach ($p in $pngs) { $bw.Write($p[1]) }
$bw.Flush()
[System.IO.File]::WriteAllBytes((Resolve-Path 'build\windows\icon.ico'), $ms.ToArray())
$bw.Dispose(); $ms.Dispose()
```

### 3) 验证

```powershell
# 确认透明：读取 ico 内嵌 PNG 的角像素 alpha 应为 0
Add-Type -AssemblyName System.Drawing
$b = [System.IO.File]::ReadAllBytes((Resolve-Path 'build\windows\icon.ico'))
$ms = New-Object System.IO.MemoryStream(,$b); $br = New-Object System.IO.BinaryReader($ms)
[void]$br.ReadUInt16(); [void]$br.ReadUInt16(); $cnt = $br.ReadUInt16()
$off = 0; $len = 0
for ($i=0; $i -lt $cnt; $i++) {
  $w = $br.ReadByte(); $h = $br.ReadByte(); [void]$br.ReadBytes(2)
  [void]$br.ReadUInt16(); [void]$br.ReadUInt16()
  $len = $br.ReadUInt32(); $off = $br.ReadUInt32()
}
$br.Dispose(); $ms.Dispose()
$png = New-Object byte[] $len; [Array]::Copy($b, $off, $png, 0, $len)
$tmp = 'build\_v.png'; [System.IO.File]::WriteAllBytes($tmp, $png)
$img = [System.Drawing.Bitmap]::FromFile($tmp)
Write-Output ("256 entry: {0}x{1}, corner alpha={2}" -f $img.Width, $img.Height, ($img.GetPixel(0,0).A))
$img.Dispose(); Remove-Item $tmp -Force
```

## 关键注意事项

- **不要 `Clear(White)` / 填充背景色**：源图透明，生成时必须用 `SourceCopy` 合成并保留 alpha，否则会把透明背景涂成白色。
- 应用运行时的内存图标缓存（`App.iconCache`）重启即清空，无需手动处理；磁盘图标缓存为 `go-launcher-data/icons/`（仅本机运行时数据，不入库）。
- 更换 exe 图标后，若桌面/任务栏仍显示旧图标，刷新 Windows 图标缓存：

  ```powershell
  Remove-Item "$env:LOCALAPPDATA\IconCache.db" -Force -ErrorAction SilentlyContinue
  Remove-Item "$env:LOCALAPPDATA\Microsoft\Windows\Explorer\iconcache_*.db" -Force -ErrorAction SilentlyContinue
  Stop-Process -Name explorer -Force
  Start-Process explorer
  ```

## 让新图标生效

图标资源由 `wails build` 打包进 exe：

```powershell
wails build
```

## 仓库提交约定

- **提交**：`build/appicon.png`、`build/windows/`（`icon.ico`、`info.json`、`wails.exe.manifest`）—— 图标源与打包配置是源码资产。
- **不提交**：`build/bin/`（编译产物）、`go-launcher-data/`（运行时数据）—— 见 `.gitignore`。
