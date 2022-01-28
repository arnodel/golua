package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/arnodel/edit"
)

//go:embed init.lua
var defaultInitLua []byte

func main() {
	flag.Usage = usage
	dumpAtEnd := false
	flag.BoolVar(&dumpAtEnd, "d", false, "Dump session to terminal when quitting")
	flag.Parse()

	// Log to a file because the terminal is used
	// f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	f, err := ioutil.TempFile("", "golua-repl.logs.")
	if err != nil {
		log.Fatalf("error opening file: %s", err)
	}
	defer f.Close()
	log.Printf("Logging to %s", f.Name())
	log.SetOutput(f)

	// Initialise the app
	buf := NewLuaBuffer()
	win := edit.NewWindow(buf)
	if dumpAtEnd {
		defer buf.WriteTo(os.Stdout)
	}

	app := edit.NewApp(win)
	if err := app.InitLuaCode("[builtin init]", defaultInitLua); err != nil {
		log.Fatalf("error in init file: %s", err)
	}

	// Initialise the screen
	screen, err := edit.NewScreen()
	if err != nil {
		log.Fatalf("error getting screen: %s", err)
	}
	defer screen.Cleanup()

	// Event loop
	for app.Running() {
		app.Draw(screen)
		app.HandleEvent(screen.PollEvent())
	}
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Help for %s:\n%s\nUsage:\n", os.Args[0], helpMessage)
	flag.PrintDefaults()
}

const helpMessage = `
golua-repl is a REPL for golua.  It lets you run lua statements interactively.

Highlights:
  - Type a Lua chunk, press [Enter] twice to execute it.
  - Quit with [Ctrl-D]
  - You can also type an expression and it will be evaluated
  - The last result is available via _ (_2, _3 fi there are multiple values)
  - Use the mouse / scrollwheel to navigate
  - Select a region with the mouse, use system clipboard to paste
  - Press [Enter] when on a previously executed statement to edit a copy of it
`
