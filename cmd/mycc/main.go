package main

import (
	"github.com/xwhiz/compiler/internal/driver"
	"os"
)

func main() {
	os.Exit(driver.Run(os.Args[1:], os.Stdout, os.Stderr))
}
