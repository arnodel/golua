package runtime

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestMarshalConst(t *testing.T) {
	type args struct {
		c      Value
		budget uint64
	}
	tests := []struct {
		name     string
		args     args
		wantUsed uint64
		// wantW    string
		wantErr bool
	}{
		{
			name: "marshal the wrong type",
			args: args{
				c: FunctionValue(nil),
			},
			wantErr: true,
		},
		{
			name: "use the budget",
			args: args{
				c:      IntValue(123456),
				budget: 1,
			},
			wantUsed: 1,
		},
		{
			name: "within budget",
			args: args{
				c:      IntValue(1234566),
				budget: 10,
			},
			wantUsed: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			gotUsed, err := MarshalConst(w, tt.args.c, tt.args.budget)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalConst() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUsed != tt.wantUsed {
				t.Errorf("MarshalConst() = %v, want %v", gotUsed, tt.wantUsed)
			}
			// if gotW := w.String(); gotW != tt.wantW {
			// 	t.Errorf("MarshalConst() = %v, want %v", gotW, tt.wantW)
			// }
		})
	}
}

func TestUnmarshalConst(t *testing.T) {
	type args struct {
		r      io.Reader
		budget uint64
	}
	tests := []struct {
		name     string
		args     args
		wantV    Value
		wantUsed uint64
		wantErr  bool
	}{
		{
			name: "consume the budget",
			args: args{
				r:      bytes.NewBuffer([]byte{6, 0, 4, byte(StringType), 1, 1, 1, 1, 1, 1, 1, 1}), // would be very long
				budget: 1000,
			},
			wantUsed: 1000,
		},
		{
			name: "wrong prefix",
			args: args{
				r:      bytes.NewBuffer([]byte{6, 1, 4, byte(StringType), 1, 1, 1, 1, 1, 1, 1, 1}), // would be very long
				budget: 1000,
			},
			wantErr: true,
		},

		{
			name: "read wrong type",
			args: args{
				r: bytes.NewBuffer([]byte{6, 0, 4, byte(FunctionType)}),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotV, gotUsed, err := UnmarshalConst(tt.args.r, tt.args.budget)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalConst() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotV, tt.wantV) {
				t.Errorf("UnmarshalConst() gotV = %v, want %v", gotV, tt.wantV)
			}
			if gotUsed != tt.wantUsed {
				t.Errorf("UnmarshalConst() gotUsed = %v, want %v", gotUsed, tt.wantUsed)
			}
		})
	}
}
