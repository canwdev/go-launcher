$ErrorActionPreference = 'Stop'

# Production build via the Wails CLI (embeds frontend assets, sets the Windows GUI
# subsystem and embeds the app icon). Requires the Wails CLI:
#   go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails build -clean

# Keep a copy at the project root, matching the previous Fyne build location.
Copy-Item -Path "build/bin/go-launcher.exe" -Destination "go-launcher.exe" -Force
