//go:build !profile
// +build !profile

package main

import (
	"flag"
	"os"
)

func main() {
	cmd := new(luaCmd)
	cmd.setFlags()
	flag.Parse()
	os.Exit(cmd.run())
}
