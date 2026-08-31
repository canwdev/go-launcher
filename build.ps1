$ErrorActionPreference = 'Stop'

# Production build: Windows GUI subsystem (no console window), release flags.
go build -trimpath -ldflags "-s -w -H windowsgui" -o go-launcher.exe .
