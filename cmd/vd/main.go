package main

import (
	"fmt"
	"os"
	"uaa/vdnd/internal/cli"
)

func main() {
	stdout, exitCode := cli.Run(os.Args[1:], cli.DefaultDeps())
	if stdout != "" {
		fmt.Println(stdout)
	}
	os.Exit(exitCode)
}
