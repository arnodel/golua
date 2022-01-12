package runtime

import (
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
