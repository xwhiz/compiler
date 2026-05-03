package main

import (
	"os"

	"github.com/xwhiz/compiler/internal/runtimecli"
)

func main() {
	os.Exit(runtimecli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
