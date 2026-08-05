package main

import (
	"fmt"
	"os"

	"github.com/1351468279/aiengine-switch/internal/app"
)

var version = "dev"

func main() {
	if err := app.Run(os.Args[1:], version); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
