package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"go/scanner"
	"go/token"
	"hash"
	"os"
	"testing"
)

// The files that define the format of marshalled values.  The run loop is
// not among them: a change to what an opcode means is up to its author to
// notice.
var marshalFormatFiles = []string{
	"../code/instructions.go",
	"../code/opcodes.go",
	"../code/reg.go",
	"loadunit.go",
	"marshal.go",
	"valuetype.go",
}

const marshalFormatHash = "68d7fe0c5db4a24eb6c0c0999c91ff16077ad551d261a03dd22aa11f8844482e"

// TestMarshalFormat is a reminder to bump marshalFormat: it fails when the
// code in the files that define the format changes.  Comments and layout
// don't count.
func TestMarshalFormat(t *testing.T) {
	h := sha256.New()
	for _, name := range marshalFormatFiles {
		hashTokens(t, h, name)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != marshalFormatHash {
		t.Fatalf(`The code that defines the format of binary chunks has changed.
If the change alters what a binary chunk holds or how it runs, and
marshalFormat (%d) is the same as in the last release (see e.g.
git show $(git describe --tags --abbrev=0):runtime/marshal.go), bump it.
Then set marshalFormatHash to %q.`, marshalFormat, got)
	}
}

// hashTokens writes the tokens of a Go file to h, without its comments.
func hashTokens(t *testing.T, h hash.Hash, name string) {
	src, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var s scanner.Scanner
	s.Init(fset.AddFile(name, -1, len(src)), src, nil, 0)
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		// A semicolon's literal says whether it was written or implied by a
		// newline, which is layout.
		if tok == token.SEMICOLON {
			lit = ""
		}
		h.Write([]byte(tok.String() + " " + lit + "\n"))
	}
	if s.ErrorCount > 0 {
		t.Fatalf("%s: %d scanning errors", name, s.ErrorCount)
	}
}
