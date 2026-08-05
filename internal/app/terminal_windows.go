//go:build windows

package app

import (
	"os"

	"golang.org/x/sys/windows"
)

func openTerminalInput() (*os.File, error) {
	name, err := windows.UTF16PtrFromString("CONIN$")
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), "CONIN$"), nil
}

func readTerminalPassword(input *os.File) ([]byte, error) {
	handle := windows.Handle(input.Fd())
	var originalMode uint32
	if err := windows.GetConsoleMode(handle, &originalMode); err != nil {
		return nil, err
	}
	if err := windows.SetConsoleMode(handle, originalMode&^windows.ENABLE_ECHO_INPUT); err != nil {
		return nil, err
	}
	defer windows.SetConsoleMode(handle, originalMode)

	buffer := make([]uint16, 64*1024+2)
	var read uint32
	if err := windows.ReadConsole(handle, &buffer[0], uint32(len(buffer)), &read, nil); err != nil {
		return nil, err
	}
	return []byte(windows.UTF16ToString(buffer[:read])), nil
}
