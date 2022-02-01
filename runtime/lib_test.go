package runtime

import (
	"os"
	"testing"

	"github.com/arnodel/golua/scanner"
)

func TestRuntime_CompileLuaChunkOrExp(t *testing.T) {
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
			_, _, err := r.CompileLuaChunkOrExp(tt.args.name, tt.args.source, tt.args.scannerOptions...)
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
