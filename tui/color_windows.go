package tui

import (
	"os"

	"golang.org/x/sys/windows"
)

// init enables ANSI/VT processing on Windows so escape sequences render
// correctly in Windows Terminal, cmd.exe, and PowerShell.
func init() {
	enableVT(os.Stdout)
	enableVT(os.Stderr)
}

func enableVT(f *os.File) {
	handle := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return
	}
	_ = windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
