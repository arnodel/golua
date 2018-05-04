package luatests

import (
	"bytes"
	"fmt"
	"regexp"
)

var expectedPtn = regexp.MustCompile(`(?m)^ *--> [=~].*$`)

type LineChecker interface {
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
	expected := expectedPtn.FindAll(source, -1)
	checkers := make([]LineChecker, len(expected))
	for i, l := range expected {
		l = bytes.TrimLeft(l, " ")
		switch l[4] {
		case '=':
			checkers[i] = LiteralLineChecker(l[5:])
		case '~':
			checkers[i] = (*RegexLineChecker)(regexp.MustCompile(string(l[5:])))
		default:
			panic("We shouldn't get there")
		}
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
			return err
		}
	}
	if len(checkers) > len(lines) {
		return fmt.Errorf("Expected %d output lines, got %d", len(checkers), len(lines))
	}
	return nil
}
