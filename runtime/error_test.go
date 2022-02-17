package runtime

import (
	"errors"
	"reflect"
	"testing"
)

func TestAsError(t *testing.T) {
	tests := []struct {
		name      string
		arg       error
		wantRtErr *Error
		wantOk    bool
	}{
		{
			name:      "nil",
			arg:       nil,
			wantRtErr: nil,
			wantOk:    false,
		},
		{
			name:      "*Error",
			arg:       NewError(StringValue("hello")),
			wantRtErr: NewError(StringValue("hello")),
			wantOk:    true,
		},
		{
			name:      "string error",
			arg:       errors.New("hello"),
			wantRtErr: nil,
			wantOk:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRtErr, gotOk := AsError(tt.arg)
			if !reflect.DeepEqual(gotRtErr, tt.wantRtErr) {
				t.Errorf("AsError() gotRtErr = %v, want %v", gotRtErr, tt.wantRtErr)
			}
			if gotOk != tt.wantOk {
				t.Errorf("AsError() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestToError(t *testing.T) {
	tests := []struct {
		name string
		arg  error
		want *Error
	}{
		{
			name: "nil",
			arg:  nil,
			want: nil,
		},
		{
			name: "nil *Error",
			arg:  (*Error)(nil),
			want: nil,
		},
		{
			name: "non nil *Error",
			arg:  errors.New("hello"),
			want: NewError(StringValue("hello")),
		},
		{
			name: "string error",
			arg:  errors.New("hi"),
			want: NewError(StringValue("hi")),
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToError(tt.arg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToError() = %v, want %v", got, tt.want)
			}
		})
	}
}
