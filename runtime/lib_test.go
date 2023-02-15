package runtime

import (
	"os"
	"testing"

	"github.com/arnodel/golua/scanner"
)

func TestRuntime_CompileAndLoadLuaChunkOrExp(t *testing.T) {
	r := New(os.Stdout)
	type args struct {
		name           string
		source         []byte
		scannerOptions []scanner.Option
	}
	tests := []struct {
		name              string
		args              args
		wantErr           bool
		wantUnexpectedEOF bool
	}{
		{
			name: "a valid expresssion",
			args: args{
				name:   "exp",
				source: []byte("1+1"),
			},
		},
		{
			name: "a valid chunk",
			args: args{
				name:   "chunk",
				source: []byte("f()\nprint('hello')"),
			},
		},
		{
			name: "invalid chunk containing expression",
			args: args{
				name:   "nogood",
				source: []byte("x+2\nprint(z)"),
			},
			wantErr: true,
		},
		{
			name: "unfinished expresssion",
			args: args{
				name:   "uexp",
				source: []byte("x+4-"),
			},
			wantErr:           true,
			wantUnexpectedEOF: true,
		},
		{
			name: "unfinished statement",
			args: args{
				name:   "ustat",
				source: []byte("do a = 2"),
			},
			wantErr:           true,
			wantUnexpectedEOF: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.CompileAndLoadLuaChunkOrExp(tt.args.name, tt.args.source, NilValue, tt.args.scannerOptions...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Runtime.CompileLuaChunkOrExp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantUnexpectedEOF && !ErrorIsUnexpectedEOF(err) {
				t.Errorf("Runtime.CompileLuaChunkOrExp() IsUnexpectedEOF = %v, want %v", err, tt.wantUnexpectedEOF)
			}
		})
	}
}

// TestSetIndexNoNewIndex tests setting new indices of values without
// a __newindex metamethod.
func TestSetIndexNoNewIndex(t *testing.T) {
	r := New(os.Stdout)
	intValue := IntValue(42)
	meta := NewTable()
	udValue := UserDataValue(NewUserData([]int{}, meta))
	if err := SetIndex(r.MainThread(), intValue, intValue, intValue); err == nil {
		t.Error("expected error indexing int value")
	}
	if err := SetIndex(r.MainThread(), udValue, intValue, intValue); err == nil {
		t.Error("expected error indexing userdata value")
	}
}
