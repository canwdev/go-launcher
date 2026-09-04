<#requires -Version 5.1
<#
.SYNOPSIS
    从 build/appicon.svg 生成 Go Launcher 应用图标资源，并可选重建 exe。

.DESCRIPTION
    完整图标生成流程：
      1) SVG -> build/appicon.png（512x512、透明背景）
         用本机 Chrome/Edge 无头模式把 SVG 渲染到品红(#FF00FF)底，
         再对 PNG 做品红色键控(chroma-key)确定性还原透明背景。
         （不依赖 Chrome 的 --default-background-color，避免其行为不稳定。）
      2) PNG -> build/windows/icon.ico（16/24/32/48/64/128/256 多尺寸透明）
         使用 .NET System.Drawing（Windows 自带，无需额外安装）。
      3) （可选，-Build）wails build -clean 把新图标打包进 exe，
         并按项目惯例复制 exe 到项目根目录。

.PARAMETER Build
    加此开关时，在生成图标资源后执行 wails build -clean 并复制 exe 到根目录。

.EXAMPLE
    # 只重新生成 appicon.png 和 icon.ico
    .\docs\generate-icon.ps1

.EXAMPLE
    # 重新生成图标资源并重建 exe
    .\docs\generate-icon.ps1 -Build
#>

param(
    [switch]$Build
)

$ErrorActionPreference = 'Stop'

# 项目根目录 = docs 的上一级
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$svg = Join-Path $root 'build\appicon.svg'
$png = Join-Path $root 'build\appicon.png'
$ico = Join-Path $root 'build\windows\icon.ico'

if (-not (Test-Path $svg)) {
    throw "未找到源图: $svg（请在 build\appicon.svg 放置矢量图标源）"
}

# ---------- 1) SVG -> PNG（品红底渲染 + 品红色键控还原透明） ----------
function Find-RenderBrowser {
    $candidates = @(
        "$env:ProgramFiles\Google\Chrome\Application\chrome.exe",
        "${env:ProgramFiles(x86)}\Google\Chrome\Application\chrome.exe",
        "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe",
        "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe"
    )
    foreach ($c in $candidates) {
        if (Test-Path $c) { return $c }
    }
    throw '未找到 Chrome/Edge 浏览器，无法将 SVG 渲染为 PNG。'
}

$browser = Find-RenderBrowser
$g = [guid]::NewGuid().ToString('N')
# HTML 与渲染产物都用「全新唯一文件名」：避免 Chrome 对 file:// 同名页面的磁盘缓存，
# 也避免与已存在的 appicon.png 产生写锁竞争（0x20 共享冲突会保留旧文件）。
$htmlPath  = Join-Path $root ("build\_appicon_render_" + $g + ".html")
$renderTmp = Join-Path $root ("build\_appicon_render_" + $g + ".png")
$keyedTmp  = Join-Path $root ("build\_appicon_keyed_"  + $g + ".png")
# 把 SVG 内联进 HTML（自包含页面，避免 <img> 相对路径/SVG 缓存加载不稳定的问题）；
# body 背景固定品红 #FF00FF，SVG 透明区域在截图里呈品红。
# 注意：必须用 .NET WriteAllText 写 HTML，不能用 Set-Content——
# Set-Content 写入的文件 Chrome(原生进程) 读不到，会渲染出空白页。
$svgText = Get-Content $svg -Raw
$htmlContent = "<!DOCTYPE html><html><head><meta charset=`"utf-8`"><style>html,body{margin:0;padding:0;background:#FF00FF;overflow:hidden}</style></head><body style='margin:0;padding:0'>$svgText</body></html>"
[System.IO.File]::WriteAllText($htmlPath, $htmlContent, (New-Object System.Text.UTF8Encoding($false)))

# 页面背景固定品红；不要依赖 --default-background-color（行为不稳定）。
# screenshot 必须用绝对路径（正斜杠），Chrome 无法写相对路径。
# 注意：这里不要用 try/finally 立即删除 html —— 若本机已有 Chrome 在运行，
# chrome.exe 启动器会把请求转交给已运行实例并提前退出，& 立刻返回，
# finally 会赶在实例真正加载 html 之前把它删掉，导致渲染出空白页。
& $browser --headless --disable-gpu --screenshot="$($renderTmp.Replace('\','/'))" --window-size=512,512 --force-device-scale-factor=1 --virtual-time-budget=3000 "file:///$($htmlPath.Replace('\','/'))"

# 轮询等待 Chrome 渲染产物出现（Chrome 写真实磁盘，沙箱视图可能有同步延迟）
$seen = $false
for ($i = 0; $i -lt 30; $i++) {
    if (Test-Path $renderTmp) { $seen = $true; break }
    Start-Sleep -Milliseconds 500
}
# 渲染已确认完成，此时才能安全删除 html
Remove-Item $htmlPath -Force -ErrorAction SilentlyContinue
if (-not $seen) {
    throw "SVG -> PNG 渲染失败：未生成 $renderTmp"
}

# 品红色键控：红≈255 且 蓝≈255 的像素，其 alpha = 绿色分量
# （品红底 -> alpha=0 透明；白色矩形 -> alpha=255 不透明；边缘混合 -> 半透明）
# 关键：先画进 Format32bppArgb 的新位图再键控保存——直接对 24bpp 源位图
# LockBits 成 32bpp 只是临时视图，Save(Png) 仍按 24bpp 编码会丢 alpha。
Add-Type -AssemblyName System.Drawing
$srcBmp = [System.Drawing.Bitmap]::FromFile((Resolve-Path $renderTmp))
$bmp = New-Object System.Drawing.Bitmap($srcBmp.Width, $srcBmp.Height, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
$gg = [System.Drawing.Graphics]::FromImage($bmp)
$gg.DrawImage($srcBmp, 0, 0, $srcBmp.Width, $srcBmp.Height)
$gg.Dispose()
$srcBmp.Dispose()
$rc = New-Object System.Drawing.Rectangle(0, 0, $bmp.Width, $bmp.Height)
$data = $bmp.LockBits($rc, [System.Drawing.Imaging.ImageLockMode]::ReadWrite, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
$stride = $data.Stride
$bytes = New-Object byte[] ($stride * $bmp.Height)
[System.Runtime.InteropServices.Marshal]::Copy($data.Scan0, $bytes, 0, $bytes.Length)
for ($y = 0; $y -lt $bmp.Height; $y++) {
    $row = $y * $stride
    for ($x = 0; $x -lt $bmp.Width; $x++) {
        $i = $row + $x * 4   # 32bppArgb: B,G,R,A
        $B = $bytes[$i]; $G = $bytes[$i + 1]; $R = $bytes[$i + 2]
        if ($R -gt 248 -and $B -gt 248) {
            $a = $G
            if ($a -gt 255) { $a = 255 }
            $bytes[$i] = 255; $bytes[$i + 1] = 255; $bytes[$i + 2] = 255; $bytes[$i + 3] = $a
        } else {
            $bytes[$i + 3] = 255
        }
    }
}
[System.Runtime.InteropServices.Marshal]::Copy($bytes, 0, $data.Scan0, $bytes.Length)
$bmp.UnlockBits($data)
$bmp.Save($keyedTmp, [System.Drawing.Imaging.ImageFormat]::Png)   # 32bppArgb 位图 -> PNG 保留 alpha
$bmp.Dispose()

# 复制到最终位置（带重试，容忍目标被瞬时占用）
$copied = $false
for ($i = 0; $i -lt 5 -and -not $copied; $i++) {
    try {
        Copy-Item -Path $keyedTmp -Destination $png -Force -ErrorAction Stop
        $copied = $true
    } catch {
        Start-Sleep -Milliseconds 1000
    }
}
if (-not $copied) {
    throw "无法写入 $png（文件被占用）：$($_.Exception.Message)"
}
Remove-Item $renderTmp, $keyedTmp -Force -ErrorAction SilentlyContinue
Write-Output "生成: $png（已还原透明背景）"

# ---------- 2) PNG -> 多尺寸透明 ICO（System.Drawing） ----------
$src = [System.Drawing.Bitmap]::FromFile((Resolve-Path $png))
$sizes = @(16, 24, 32, 48, 64, 128, 256)
$pngs = New-Object System.Collections.ArrayList
foreach ($s in $sizes) {
    $bmp = New-Object System.Drawing.Bitmap($s, $s, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
    $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
    $g.CompositingMode = [System.Drawing.Drawing2D.CompositingMode]::SourceCopy   # 保留 alpha，勿填充底色
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
[System.IO.File]::WriteAllBytes($ico, $ms.ToArray())
$bw.Dispose(); $ms.Dispose()
Write-Output "生成: $ico ($count 个尺寸: $($sizes -join '/'))"

# ---------- 3) 验证透明：读取 ico 内嵌 256 条目角像素 alpha 应为 0 ----------
$b = [System.IO.File]::ReadAllBytes($ico)
$ms = New-Object System.IO.MemoryStream(,$b); $br = New-Object System.IO.BinaryReader($ms)
[void]$br.ReadUInt16(); [void]$br.ReadUInt16(); $cnt = $br.ReadUInt16()
$off = 0; $len = 0
for ($i = 0; $i -lt $cnt; $i++) {
    $w = $br.ReadByte(); $h = $br.ReadByte(); [void]$br.ReadBytes(2)
    [void]$br.ReadUInt16(); [void]$br.ReadUInt16()
    $len = $br.ReadUInt32(); $off = $br.ReadUInt32()
}
$br.Dispose(); $ms.Dispose()
$png = New-Object byte[] $len; [Array]::Copy($b, $off, $png, 0, $len)
$tmp = Join-Path $root 'build\_v.png'; [System.IO.File]::WriteAllBytes($tmp, $png)
$img = [System.Drawing.Bitmap]::FromFile($tmp)
$alpha = $img.GetPixel(0, 0).A
Write-Output ("验证: 256 条目 {0}x{1}, 角像素 alpha={2} {3}" -f $img.Width, $img.Height, $alpha, $(if ($alpha -eq 0) { '(OK 透明)' } else { '(警告: 非透明!)' }))
$img.Dispose(); Remove-Item $tmp -Force -ErrorAction SilentlyContinue

# ---------- 4) （可选）重建 exe ----------
if ($Build) {
    Write-Output '执行 wails build -clean ...'
    wails build -clean
    Copy-Item -Path 'build\bin\go-launcher.exe' -Destination 'go-launcher.exe' -Force
    Write-Output '完成: exe 已重建并复制到项目根目录。'
} else {
    Write-Output '完成（未重建 exe）。需要打包请加 -Build。'
}
