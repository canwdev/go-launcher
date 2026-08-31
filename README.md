# go-launcher

A minimal [Fyne](https://fyne.io/) demo app for Go. It opens a window with a button; clicking the button prints `Hello!` to the console.

## Requirements

- Go 1.23 or later
- A C compiler (for Fyne's native dependencies):
  - **Windows**: GCC via [MinGW-w64](https://www.mingw-w64.org/) / [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) (installed on `PATH`)
  - **macOS**: Xcode Command Line Tools
  - **Linux**: `gcc` and the GTK development headers (`libgtk-3-dev` on Debian/Ubuntu)

## Run

```sh
go run .
```

## Build

```sh
go build
```

## Overview

`main.go`:

```go
a := app.New()                          // create the application
w := a.NewWindow("Hello")               // create a window titled "Hello"

w.SetContent(widget.NewButton(          // add a button as the window content
    "Click me",
    func() { println("Hello!") },       // callback when clicked
))

w.ShowAndRun()                          // show the window and run the event loop
```

## References

- [Fyne documentation](https://docs.fyne.io/)
- [Fyne on GitHub](https://github.com/fyne-io/fyne)
