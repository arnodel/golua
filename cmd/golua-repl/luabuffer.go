package main

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/arnodel/edit"
	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luastrings"
	"github.com/arnodel/golua/runtime"
)

type luaLineType uint8

const (
	luaComment luaLineType = iota
	luaInput
	luaStdout
	luaOutput
	luaParseError
	luaRuntimeError
)

type luaLineData struct {
	index int
	tp    luaLineType
}

type LuaBuffer struct {
	r            *runtime.Runtime
	stdout       bytes.Buffer
	buf          edit.FileBuffer
	currentIndex int
}

var _ edit.Buffer = (*LuaBuffer)(nil)

func NewLuaBuffer() *LuaBuffer {
	b := &LuaBuffer{}
	b.r = runtime.New(&b.stdout)
	lib.LoadAll(b.r)
	b.buf.AppendLine(edit.NewLineFromString("Welcome to Golua REPL! Press [Enter] twice to run code, [Ctrl-D] to quit.", luaLineData{tp: luaComment}))
	b.buf.AppendLine(edit.NewLineFromString(inputName(0), luaLineData{tp: luaComment}))
	b.buf.AppendLine(edit.NewLineFromString("> ", luaLineData{tp: luaInput}))
	return b
}

func (b *LuaBuffer) LineCount() int {
	return b.buf.LineCount()
}

func (b *LuaBuffer) GetLine(l, c int) (edit.Line, error) {
	return b.buf.GetLine(l, c)
}

func (b *LuaBuffer) InsertRune(r rune, l, c int) error {
	_, err := b.getEditableLine(l, c)
	if err != nil {
		return err
	}
	return b.buf.InsertRune(r, l, c)
}

// When pasting code, remove "> " at the start of lines.
var promptPtn = regexp.MustCompile(`(?m)^> `)

func (b *LuaBuffer) InsertString(s string, l, c int) (int, int, error) {
	// s = promptPtn.ReplaceAllLiteralString(s, "")
	return b.buf.InsertString(s, l, c)
}

func (b *LuaBuffer) InsertLine(l int, line edit.Line) error {
	if l != b.buf.LineCount() {
		_, err := b.getEditableLine(l, -1)
		if err != nil && l != 0 {
			_, err = b.getEditableLine(l-1, -1)
		}
		if err != nil {
			return err
		}
	}
	data := luaLineData{index: b.currentIndex, tp: luaInput}
	line = edit.NewLineFromString("> ", data).MergeWith(line)

	return b.buf.InsertLine(l, line)
}

func (b *LuaBuffer) DeleteLine(l int) error {
	if _, err := b.getEditableLine(l, -1); err != nil {
		return err
	}
	return b.buf.DeleteLine(l)
}

func (b *LuaBuffer) AppendLine(line edit.Line) {
	_ = b.InsertLine(b.LineCount(), line)
}

func (b *LuaBuffer) SplitLine(l, c int) error {
	line, err := b.getEditableLine(l, c)
	if err != nil {
		return err
	}
	l1, l2 := line.SplitAt(c)
	if err := b.buf.SetLine(l, l1); err != nil {
		return err
	}
	return b.InsertLine(l+1, l2)
}

func (b *LuaBuffer) AdvancePos(l, c, dl, dc int) (int, int) {
	l, c = b.buf.AdvancePos(l, c, dl, dc)
	if c < 2 {
		line := b.buf.Line(l)
		data := line.Meta.(luaLineData)
		if data.tp == luaInput {
			if dc >= 0 || l == 0 {
				c = 2
			} else {
				l--
				c = b.buf.Line(l).Len()
			}
		}
	}
	return l, c
}

func (b *LuaBuffer) DeleteRuneAt(l, c int) error {
	_, err := b.getEditableLine(l, c)
	if err != nil {
		return err
	}
	return b.buf.DeleteRuneAt(l, c)
}

func (b *LuaBuffer) EndPos() (int, int) {
	return b.buf.EndPos()
}

func (b *LuaBuffer) MergeLineWithPrevious(l int) error {
	l2, err := b.getEditableLine(l, -1)
	if err != nil {
		return err
	}
	_, err = b.getEditableLine(l-1, -1)
	if err != nil {
		return err
	}
	l2.Runes = l2.Runes[2:]
	b.buf.SetLine(l, l2)
	return b.buf.MergeLineWithPrevious(l)
}

func (b *LuaBuffer) Save() error {

	return fmt.Errorf("unimplemented")
}

func (b *LuaBuffer) WriteTo(w io.Writer) (int64, error) {
	n := 0
	end := b.LineCount()
	start := 0
	if end >= 1000 {
		start = end - 1000
	}
	for i := start; i < end; i++ {
		l := b.buf.Line(i)
		ln, err := fmt.Fprintf(w, "%s\n", l.String())
		n += ln
		if err != nil {
			return int64(n), err
		}
	}
	return int64(n), nil
}

func (b *LuaBuffer) StyledLineIter(l, c int) edit.StyledLineIter {
	line := b.buf.Line(l)
	meta := line.Meta.(luaLineData)
	style := edit.DefaultStyle
	switch meta.tp {
	case luaInput:
	case luaStdout:
		style = style.Foreground(edit.ColorYellow)
	case luaOutput:
		style = style.Foreground(edit.ColorGreen).Bold(true)
	case luaParseError:
		style = style.Foreground(edit.ColorRed).Bold(true)
	case luaRuntimeError:
		style = style.Foreground(edit.ColorRed).Bold(true)
	case luaComment:
		style = style.Bold(true)
	}
	return edit.NewConstStyleLineIter(line.Iter(c), style)
}

func (b *LuaBuffer) Kind() string {
	return "luarepl"
}

func (b *LuaBuffer) StringFromRegion(l0, c0, l1, c1 int) (string, error) {
	return b.buf.StringFromRegion(l0, c0, l1, c1)
}

func (b *LuaBuffer) getEditableLine(l, c int) (edit.Line, error) {
	line, err := b.buf.GetLine(l, c)
	if err != nil {
		return edit.Line{}, err
	}
	data := line.Meta.(luaLineData)
	if data.index != b.currentIndex || data.tp != luaInput || (c >= 0 && c < 2) {
		return edit.Line{}, fmt.Errorf("read only")
	}
	return line, nil
}

func (b *LuaBuffer) getChunk(l int) (string, error) {
	line, err := b.getEditableLine(l, -1)
	if err != nil {
		return "", err
	}
	index := line.Meta.(luaLineData).index
	meta := luaLineData{index: index, tp: luaInput}
	for l > 0 {
		if b.buf.Line(l-1).Meta != meta {
			break
		}
		l--
	}
	var builder strings.Builder
	for {
		builder.WriteString(string(b.buf.Line(l).Runes[2:]))
		builder.WriteByte('\n')
		l++
		if l >= b.buf.LineCount() || b.buf.Line(l).Meta != meta {
			break
		}
	}
	return builder.String(), nil
}

func (b *LuaBuffer) RunCurrent() error {
	b.stdout.Reset()
	l := b.buf.LineCount()
	for ; l > 0; l-- {
		if b.buf.Line(l-1).Meta.(luaLineData).tp != luaParseError {
			break
		}
	}
	b.buf.Truncate(l)
	chunk, err := b.getChunk(b.LineCount() - 1)
	if err != nil {
		return err
	}
	if strings.TrimSpace(chunk) == "" {
		return nil
	}
	clos, err := b.r.CompileAndLoadLuaChunkOrExp(inputName(b.currentIndex), []byte(chunk), runtime.TableValue(b.r.GlobalEnv()))
	if err != nil {
		b.buf.AppendLine(edit.NewLineFromString(err.Error(), luaLineData{index: b.currentIndex, tp: luaParseError}))
		return err
	}

	term := runtime.NewTerminationWith(nil, 0, true)
	rtErr := runtime.Call(b.r.MainThread(), runtime.FunctionValue(clos), nil, term)

	// Print stdout
	meta := luaLineData{index: b.currentIndex, tp: luaStdout}
	for {
		line, err := b.stdout.ReadString('\n')
		if err != nil {
			break
		}
		b.buf.AppendLine(edit.NewLineFromString(line[:len(line)-1], meta))
	}

	// Print runtime error
	if rtErr != nil {
		meta.tp = luaRuntimeError
		for _, s := range strings.Split(rtErr.Error(), "\n") {
			b.buf.AppendLine(edit.NewLineFromString("! "+s, meta))
		}
	}

	// Print result
	meta.tp = luaOutput
	for i, x := range term.Etc() {
		b.r.SetEnv(b.r.GlobalEnv(), fmt.Sprintf("_%d", i+1), x)
		if i == 0 {
			b.r.SetEnv(b.r.GlobalEnv(), "_", x)
		}
		b.buf.AppendLine(edit.NewLineFromString("= "+quoteLuaVal(x), meta))
	}

	b.currentIndex++
	b.buf.AppendLine(edit.NewLineFromString(inputName(b.currentIndex), luaLineData{index: b.currentIndex, tp: luaComment}))
	b.buf.AppendLine(edit.NewLineFromString("> ", luaLineData{index: b.currentIndex, tp: luaInput}))
	return nil
}

func (b *LuaBuffer) IsCurrentLast(l int) bool {
	line, err := b.getEditableLine(l, -1)
	if err != nil || line.Len() > 2 {
		return false
	}
	_, err = b.getEditableLine(l+1, -1)
	return err != nil
}

func (b *LuaBuffer) deleteCurrent() {
	l := b.buf.LineCount()
	for ; l > 0; l-- {
		data := b.buf.Line(l - 1).Meta.(luaLineData)
		if data.index != b.currentIndex || (data.tp != luaParseError && data.tp != luaInput) {
			break
		}
	}
	b.buf.Truncate(l)
}

func (b *LuaBuffer) ResetCurrent() {
	b.deleteCurrent()
	b.buf.AppendLine(edit.NewLineFromString("> ", luaLineData{index: b.currentIndex, tp: luaInput}))
}

func (b *LuaBuffer) CopyToCurrent(l int) error {
	if l < 0 || l >= b.LineCount() {
		return fmt.Errorf("out of range")
	}
	line := b.buf.Line(l)
	meta := line.Meta.(luaLineData)
	if meta.tp != luaInput {
		return fmt.Errorf("can only copy input")
	}
	if meta.index == b.currentIndex {
		return fmt.Errorf("can only copy previous input")
	}
	for ; l > 0; l-- {
		prevMeta := b.buf.Line(l - 1).Meta.(luaLineData)
		if prevMeta != meta {
			break
		}
	}
	b.deleteCurrent()
	currentMeta := luaLineData{index: b.currentIndex, tp: luaInput}
	for {
		line := b.buf.Line(l)
		if line.Meta.(luaLineData) != meta {
			return nil
		}
		b.buf.AppendLine(edit.NewLineFromString(line.String(), currentMeta))
		l++
	}
}

func quoteLuaVal(v runtime.Value) string {
	s, ok := v.TryString()
	if ok {
		var q byte = '"'
		if strings.ContainsRune(s, '"') {
			q = '\''
		}
		return luastrings.Quote(s, q)
	}
	s, _ = v.ToString()
	return s
}

func inputName(n int) string {
	return fmt.Sprintf("[%d]", n+1)
}
