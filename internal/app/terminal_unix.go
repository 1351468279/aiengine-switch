//go:build !windows

package app

import (
	"os"

	"golang.org/x/term"
)

func openTerminalInput() (*os.File, error) {
	return os.Open("/dev/tty")
}

func readTerminalPassword(input *os.File) ([]byte, error) {
	return term.ReadPassword(int(input.Fd()))
}
