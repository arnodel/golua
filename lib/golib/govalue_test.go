package golib

import (
	"reflect"
	"testing"

	rt "github.com/arnodel/golua/runtime"
)

func Test_reflectToValue(t *testing.T) {
	meta := rt.NewTable()
	testTable := rt.NewTable()
	testTable.Set(rt.String("key"), rt.Int(123))
	testUdata := rt.NewUserData(1.2, meta)
	testStruct := struct {
		Foo int
		Bar string
	}{Foo: 2, Bar: "hi"}
	tests := []struct {
		name string
		arg  interface{}
		want rt.Value
	}{
		{
			name: "empty",
			arg:  nil,
			want: nil,
		},
		{
			name: "int",
			arg:  int(8),
			want: rt.Int(8),
		},
		{
			name: "int8",
			arg:  int8(-33),
			want: rt.Int(-33),
		},
		{
			name: "uint",
			arg:  uint(21),
			want: rt.Int(21),
		},
		{
			name: "uint16",
			arg:  uint16(777),
			want: rt.Int(777),
		},
		{
			name: "float64",
			arg:  float64(1.2),
			want: rt.Float(1.2),
		},
		{
			name: "string",
			arg:  string("hello"),
			want: rt.String("hello"),
		},
		{
			name: "bool",
			arg:  true,
			want: rt.Bool(true),
		},
		{
			name: "[]byte",
			arg:  []byte("bonjour"),
			want: rt.String("bonjour"),
		},
		{
			name: "lua table",
			arg:  testTable,
			want: testTable,
		},
		{
			name: "lua userdata",
			arg:  testUdata,
			want: testUdata,
		},
		{
			name: "non luable type",
			arg:  testStruct,
			want: rt.NewUserData(testStruct, meta),
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reflectToValue(reflect.ValueOf(tt.arg), meta); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reflectToValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

type tabledef map[interface{}]interface{}

func (t tabledef) table() *rt.Table {
	tbl := rt.NewTable()
	for k, v := range t {
		tbl.Set(reflectToValue(reflect.ValueOf(k), nil), reflectToValue(reflect.ValueOf(v), nil))
	}
	return tbl
}

func Test_valueToType(t *testing.T) {
	var thread *rt.Thread
	meta := rt.NewTable()
	tests := []struct {
		name    string
		v       rt.Value
		want    interface{}
		wantErr bool
	}{
		{
			name: "userdata assignable",
			v:    rt.NewUserData(int(12), meta),
			want: int(12),
		},
		{
			name: "userdata convertible",
			v:    rt.NewUserData(int8(123), meta),
			want: int(123),
		},
		{
			name:    "userdata not assignable or convertible",
			v:       rt.NewUserData(string("hello"), meta),
			want:    int(1),
			wantErr: true,
		},
		{
			name:    "pointer to non struct",
			v:       int(123),
			want:    new(int),
			wantErr: true,
		},
		{
			name: "pointer to struct that works",
			v:    tabledef{}.table(),
			want: &struct{}{},
		},
		{
			name:    "pointer to struct that doesn't work",
			v:       tabledef{"Foo": 2}.table(),
			want:    &struct{}{},
			wantErr: true,
		},
		{
			name: "struct that works",
			v:    tabledef{}.table(),
			want: struct{}{},
		},
		{
			name:    "struct that doesn't work",
			v:       tabledef{"Foo": 2}.table(),
			want:    struct{}{},
			wantErr: true,
		},
		{
			name: "rt.Int to int",
			v:    rt.Int(123),
			want: int(123),
		},
		{
			name: "integral rt.Float to int",
			v:    rt.Float(111),
			want: int(111),
		},
		{
			name: "integral string to int",
			v:    rt.String("432"),
			want: int(432),
		},
		{
			name: "rt.Float to float64",
			v:    rt.Float(1.3),
			want: float64(1.3),
		},
		{
			name: "rt.Int to float64",
			v:    rt.Int(-123),
			want: float64(-123),
		},
		{
			name: "floaty string to float64",
			v:    rt.String("3.14"),
			want: float64(3.14),
		},
		{
			name:    "table to float64",
			v:       rt.NewTable(),
			want:    float64(123),
			wantErr: true,
		},
		{
			name: "nil to bool",
			v:    nil,
			want: false,
		},
		{
			name: "rt.Int(0) to bool",
			v:    rt.Int(0),
			want: true,
		},
		{
			name: "false to bool",
			v:    rt.Bool(false),
			want: false,
		},
		{
			name: "true to bool",
			v:    rt.Bool(true),
			want: true,
		},
		{
			name: "empty string to bool",
			v:    rt.String(""),
			want: true,
		},
		{
			name: "rt.String to []byte",
			v:    rt.String("foo"),
			want: []byte("foo"),
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueToType(thread, tt.v, reflect.TypeOf(tt.want))
			if (err != nil) != tt.wantErr {
				t.Errorf("valueToType() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got.Interface(), tt.want) {
				t.Errorf("valueToType() = %v, want %v", got, tt.want)
			}
		})
	}
}
