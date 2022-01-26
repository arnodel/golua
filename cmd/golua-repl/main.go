package main

import (
	_ "embed"
	"io/ioutil"
	"log"

	"github.com/arnodel/edit"
)

//go:embed init.lua
var defaultInitLua []byte

func main() {

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
