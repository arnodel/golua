package luatesting

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
)

var expectedPtn = regexp.MustCompile(`^ *--> [=~].*$`)

type LineChecker struct {
	SourceLineno int
	lineChecker
}

type lineChecker interface {
	CheckLine([]byte) error
}

type LiteralLineChecker []byte

func (c LiteralLineChecker) CheckLine(output []byte) error {
	expected := []byte(c)
	if bytes.Equal(output, expected) {
		return nil
	}
	return fmt.Errorf("expected: %q, got: %q", expected, output)
}

type RegexLineChecker regexp.Regexp

func (c *RegexLineChecker) CheckLine(output []byte) error {
	ptn := (*regexp.Regexp)(c)
	if ptn.Match(output) {
		return nil
	}
	return fmt.Errorf("expected regex: %s, got %q", ptn, output)
}

func ExtractLineCheckers(source []byte) []LineChecker {
	var (
		scanner  = bufio.NewScanner(bytes.NewReader(source))
		lineno   = 0
		checkers []LineChecker
	)
	for scanner.Scan() {
		lineno++
		var (
			line = scanner.Bytes()
			lc   lineChecker
		)
		if !expectedPtn.Match(line) {
			continue
		}
		line = bytes.TrimLeft(line, " ")
		switch line[4] {
		case '=':
			lit := make([]byte, len(line)-5)
			copy(lit, line[5:])
			lc = LiteralLineChecker(lit)
		case '~':
			lc = (*RegexLineChecker)(regexp.MustCompile(string(line[5:])))
		default:
			panic("We shouldn't get there")
		}
		checkers = append(checkers, LineChecker{SourceLineno: lineno, lineChecker: lc})
	}
	return checkers
}

func CheckLines(output []byte, checkers []LineChecker) error {
	if len(output) > 0 && output[len(output)-1] == '\n' {
		output = output[:len(output)-1]
	}
	lines := bytes.Split(output, []byte{'\n'})
	for i, line := range lines {
		if i >= len(checkers) {
			return fmt.Errorf("Extra output line #%d: %q", i+1, line)
		}
		if err := checkers[i].CheckLine(line); err != nil {
			return fmt.Errorf("[output line %d, source line %d] %s", i+1, checkers[i].SourceLineno, err)
		}
	}
	if len(checkers) > len(lines) {
		return fmt.Errorf("Expected %d output lines, got %d", len(checkers), len(lines))
	}
	return nil
}
