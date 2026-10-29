package runtime

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func BenchmarkValue(b *testing.B) {
	for n := 0; n < b.N; n++ {
		sv := IntValue(0)
		for i := 0; i < 1000; i++ {
			iv := IntValue(int64(i))
			sv, _ = Add(sv, iv)
		}
	}
}

func BenchmarkAsCont(b *testing.B) {
	v1 := ContValue(new(GoCont))
	v2 := ContValue(new(LuaCont))
	v3 := ContValue(new(Termination))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v1.AsCont()
		_ = v2.AsCont()
		_ = v3.AsCont()
	}
}

func TestAsValue(t *testing.T) {
	tests := []struct {
		name string
		arg  interface{}
		want Value
	}{
		{
			name: "nil arg",
			want: NilValue,
		},
		{
			name: "int64 arg",
			arg:  int64(1555),
			want: IntValue(1555),
		},
		{
			name: "int arg",
			arg:  int(999),
			want: IntValue(999),
		},
		{
			name: "float64 arg",
			arg:  float64(0.9),
			want: FloatValue(0.9),
		},
		{
			name: "float32 arg",
			arg:  float32(66),
			want: FloatValue(66),
		},
		{
			name: "bool arg",
			arg:  true,
			want: BoolValue(true),
		},
		{
			name: "string arg",
			arg:  "hello",
			want: StringValue("hello"),
		},
		{
			name: "other type",
			arg:  []byte{'a'},
			want: Value{iface: []byte{'a'}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AsValue(tt.arg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AsValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValue_Interface(t *testing.T) {

	var tbl = NewTable()

	tests := []struct {
		name string
		recv Value
		want interface{}
	}{
		{
			name: "nil",
			recv: NilValue,
			want: nil,
		},
		{
			name: "int",
			recv: IntValue(50),
			want: int64(50),
		},
		{
			name: "float",
			recv: FloatValue(1.2),
			want: float64(1.2),
		},
		{
			name: "bool",
			recv: BoolValue(false),
			want: false,
		},
		{
			name: "table value",
			recv: TableValue(tbl),
			want: tbl,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.recv.Interface(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Value.Interface() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValue_ToString(t *testing.T) {

	tests := []struct {
		name string
		recv Value
		want string
		ok   bool
	}{
		{
			name: "NilValue",
			recv: NilValue,
			want: "nil",
		},
		{
			name: "IntValue",
			recv: IntValue(456),
			want: "456",
			ok:   true,
		},
		{
			name: "FloatValue",
			recv: FloatValue(-1.5),
			want: "-1.5",
			ok:   true,
		},
		{
			name: "StringValue",
			recv: StringValue("a string"),
			want: "a string",
			ok:   true,
		},
		{
			name: "TableValue",
			recv: TableValue(NewTable()),
			want: "table: 0x",
		},
		{
			name: "CodeValue",
			recv: CodeValue(nil),
			want: "code: 0x0",
		},
		{
			name: "GoFunction",
			recv: FunctionValue(NewGoFunction(nil, "foo", 0, false)),
			want: "gofunction: foo",
		},
		{
			name: "Closure",
			recv: FunctionValue((*Closure)(nil)),
			want: "function: 0x0",
		},
		{
			name: "Thread",
			recv: ThreadValue((*Thread)(nil)),
			want: "thread: 0x0",
		},
		{
			name: "UserData",
			recv: UserDataValue(NewUserData(nil, nil)),
			want: "userdata: 0x",
		},
		{
			name: "other type",
			recv: AsValue(byte(123)),
			want: "<unknown>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.recv.ToString()
			if !strings.HasPrefix(got, tt.want) {
				t.Errorf("Value.ToString() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.ok {
				t.Errorf("Value.ToString() got1 = %v, want %v", got1, tt.ok)
			}
		})
	}
}

func TestFloatToString(t *testing.T) {
	// The expected strings are the output of PUC Lua 5.5.1's tostring().
	tests := []struct {
		x    float64
		want string
	}{
		{0, "0.0"},
		{math.Copysign(0, -1), "-0.0"},
		{-1.5, "-1.5"},
		{999999, "999999.0"},
		{1234567, "1234567.0"},
		{-1234567, "-1234567.0"},
		{4294967294, "4294967294.0"},
		{1e14, "100000000000000.0"},
		{1e15, "1e+15"},
		{1e100, "1e+100"},
		{0.1, "0.1"},
		{123.456, "123.456"},
		{1e-4, "0.0001"},
		{1e-5, "1e-05"},
		{3.14159265358979, "3.14159265358979"},
		{1.0 / 3, "0.33333333333333331"},
		{0.30000000000000004, "0.30000000000000004"},
		{110.00000000000001, "110.00000000000001"},
		{1 << 53, "9007199254740992.0"},
		{1 << 63, "9.2233720368547758e+18"},
		{18446744073709551614, "1.8446744073709552e+19"},
		{math.Inf(1), "inf"},
		{math.Inf(-1), "-inf"},
		{math.NaN(), "nan"},
		{math.Copysign(math.NaN(), -1), "-nan"},
	}
	for _, tt := range tests {
		if got := FloatToString(tt.x); got != tt.want {
			t.Errorf("FloatToString(%v) = %q, want %q", tt.x, got, tt.want)
		}
	}
}

func TestValue_AsCallable(t *testing.T) {
	tests := []struct {
		name   string
		recv   Value
		want   Callable
		panics bool
	}{
		{
			name: "Closure value ok",
			recv: FunctionValue((*Closure)(nil)),
			want: (*Closure)(nil),
		},
		{
			name: "GoFunction value ok",
			recv: FunctionValue((*GoFunction)(nil)),
			want: (*GoFunction)(nil),
		},
		{
			name:   "other type panics",
			recv:   IntValue(5),
			panics: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panics {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic")
					}
				}()
			}
			if got := tt.recv.AsCallable(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Value.AsCallable() = %v, want %v", got, tt.want)
			}
		})
	}
}
