module github.com/arnodel/golua/cmd/golua-repl

go 1.17

// replace github.com/arnodel/edit => /Users/arno/Projects/edit
replace github.com/arnodel/golua => ../../

require (
	github.com/arnodel/edit v0.0.0-20220202110212-dfc8d7a13890
	github.com/arnodel/golua v0.0.0-20220201164830-478648eb3f09
)

require (
	github.com/arnodel/strftime v0.1.6 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/gdamore/tcell/v2 v2.4.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	golang.org/x/sys v0.0.0-20220128215802-99c3d69c2c27 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
)
