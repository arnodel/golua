//+build !profile

package main

import (
	"flag"
)

func main() {
	cmd := new(luaCmd)
	cmd.setFlags()
	flag.Parse()
	cmd.run()
}
