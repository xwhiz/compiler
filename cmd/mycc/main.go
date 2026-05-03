package main

import (
	"os"
	"github.com/xwhiz/compiler/internal/driver"
)

func main() {
	os.Exit(driver.Run(os.Args[1:], os.Stdout, os.Stderr))
}
