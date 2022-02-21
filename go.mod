module github.com/arnodel/golua

go 1.17

require (
	github.com/arnodel/edit v0.0.0-20220202110212-dfc8d7a13890 // Only needed when building cmd/golua-repl
	github.com/arnodel/strftime v0.1.6
)

// Indirect dependencies pulled by github.com/arnodel/edit for cmd/golua-repl,
// not used by core packages.
require (
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/gdamore/tcell/v2 v2.4.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.0.3 // indirect
	github.com/mattn/go-runewidth v0.0.10 // indirect
	github.com/rivo/uniseg v0.1.0 // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/text v0.3.6 // indirect
)
