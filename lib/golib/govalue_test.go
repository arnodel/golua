package golib

import (
	"errors"
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

func Test_fillStruct(t *testing.T) {
	type testStruct struct {
		Foo int
	}
	thread := new(rt.Thread)
	tests := []struct {
		name    string
		before  reflect.Value
		after   interface{}
		v       rt.Value
		wantErr bool
	}{
		{
			name:    "not a table",
			before:  reflect.ValueOf(struct{}{}),
			v:       rt.Int(12),
			wantErr: true,
		},
		{
			name:    "table with non-string field",
			before:  reflect.ValueOf(struct{}{}),
			v:       tabledef{10: 12}.table(),
			wantErr: true,
		},
		{
			name:   "success",
			before: reflect.ValueOf(&testStruct{}).Elem(),
			v:      tabledef{"Foo": 23}.table(),
			after:  testStruct{Foo: 23},
		},
		{
			name:    "incorrect type for field",
			before:  reflect.ValueOf(&testStruct{}).Elem(),
			v:       tabledef{"Foo": "hi"}.table(),
			wantErr: true,
		},
		{
			name:    "Non-existent field",
			before:  reflect.ValueOf(&testStruct{}).Elem(),
			v:       tabledef{"Bar": 1}.table(),
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fillStruct(thread, tt.before, tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("fillStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(tt.before.Interface(), tt.after) {
				t.Errorf("fillStruct() expected %s got %s", tt.after, tt.before.Interface())
			}
		})
	}
}

func Test_goIndex(t *testing.T) {
	thread := new(rt.Thread)
	meta := rt.NewTable()
	testErr := errors.New("hello")
	testInt := int(12)

	tests := []struct {
		name            string
		goval           interface{}
		key             rt.Value
		want            rt.Value
		wantErr         bool
		doNotCheckValue bool
	}{
		{
			name:            "method on struct pointer",
			goval:           testErr,
			key:             rt.String("Error"),
			doNotCheckValue: true,
		},
		{
			name:    "pointer to non struct",
			goval:   &testInt,
			key:     rt.String("x"),
			wantErr: true,
		},
		{
			name:    "non-string index for struct",
			goval:   struct{ Foo int }{},
			key:     rt.Bool(true),
			wantErr: true,
		},
		{
			name:    "index for struct not referring to a field",
			goval:   struct{ Foo int }{},
			key:     rt.String("Bar"),
			wantErr: true,
		},
		{
			name:  "index for struct referring to a field",
			goval: struct{ Foo int }{Foo: 12},
			key:   rt.String("Foo"),
			want:  rt.Int(12),
		},
		{
			name:    "map index of incompatible type",
			goval:   map[int]int{},
			key:     rt.String("hi"),
			wantErr: true,
		},
		{
			name:  "map index of compatible type",
			goval: map[string]int{"hi": 34},
			key:   rt.String("hi"),
			want:  rt.Int(34),
		},
		{
			name:    "non-integral slice index",
			goval:   []string{"hi", "there"},
			key:     rt.String("bad"),
			wantErr: true,
		},
		{
			name:  "integral slice index within bounds",
			goval: []string{"hi", "there"},
			key:   rt.Float(1),
			want:  rt.String("there"),
		},
		{
			name:    "integral slice index greater than length-1",
			goval:   []string{"hi", "there"},
			key:     rt.Float(2),
			wantErr: true,
		},
		{
			name:    "integral slice index negative",
			goval:   []string{"hi", "there"},
			key:     rt.Int(-1),
			wantErr: true,
		},
		{
			name:    "unsupported type (function)",
			goval:   func() {},
			key:     rt.Int(1),
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := goIndex(thread, rt.NewUserData(tt.goval, meta), tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("goIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !tt.doNotCheckValue && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("goIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_goSetIndex(t *testing.T) {
	thread := new(rt.Thread)
	meta := rt.NewTable()
	testInt := int(12)
	tests := []struct {
		name    string
		goval   interface{}
		after   interface{}
		key     rt.Value
		val     rt.Value
		wantErr bool
	}{
		{
			name:    "pointer to non struct",
			goval:   &testInt,
			key:     rt.String("key"),
			val:     rt.String("val"),
			wantErr: true,
		},
		{
			name:    "non string struct index",
			goval:   &struct{}{},
			key:     rt.Bool(true),
			val:     rt.Int(10),
			wantErr: true,
		},
		{
			name:    "non existing struct field",
			goval:   &struct{ Foo int }{},
			key:     rt.String("Bar"),
			val:     rt.Int(1),
			wantErr: true,
		},
		{
			name:    "struct field set to incompatible type",
			goval:   &struct{ Foo int }{},
			key:     rt.String("Foo"),
			val:     rt.String("hi"),
			wantErr: true,
		},
		{
			name:  "struct field set to incompatible type",
			goval: &struct{ Foo int }{},
			key:   rt.String("Foo"),
			val:   rt.Int(12),
			after: &struct{ Foo int }{Foo: 12},
		},
		{
			name:    "struct field non settable",
			goval:   struct{ Foo int }{},
			key:     rt.String("Foo"),
			val:     rt.Int(12),
			wantErr: true,
		},
		{
			name:    "map key of incompatible type",
			goval:   map[int]string{},
			key:     rt.String("three"),
			val:     rt.Int(444),
			wantErr: true,
		},
		{
			name:    "map value of incompatible type",
			goval:   map[int]string{},
			key:     rt.Int(444),
			val:     rt.Bool(false),
			wantErr: true,
		},
		{
			name:  "map success",
			goval: map[int]string{},
			key:   rt.Int(444),
			val:   rt.String("chouette"),
			after: map[int]string{444: "chouette"},
		},
		{
			name:    "non integer slice index",
			goval:   []int{3, 2, 1},
			key:     rt.String("deux"),
			val:     rt.Int(12),
			wantErr: true,
		},
		{
			name:    "negative slice index",
			goval:   []int{3, 2, 1},
			key:     rt.Int(-1),
			val:     rt.Int(12),
			wantErr: true,
		},
		{
			name:    "slice index > len-1",
			goval:   []int{3, 2, 1},
			key:     rt.Int(3),
			val:     rt.Int(12),
			wantErr: true,
		},
		{
			name:    "slice value of incompatible type",
			goval:   []int{3, 2, 1},
			key:     rt.Int(2),
			val:     rt.Bool(true),
			wantErr: true,
		},
		{
			name:  "successful slice",
			goval: []int{3, 2, 1},
			key:   rt.Int(1),
			val:   rt.Int(12),
			after: []int{3, 12, 1},
		},
		{
			name:    "unsupported go type",
			goval:   false,
			key:     rt.Int(1),
			val:     rt.Int(2),
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := rt.NewUserData(tt.goval, meta)
			err := goSetIndex(thread, u, tt.key, tt.val)
			if (err != nil) != tt.wantErr {
				t.Errorf("goSetIndex() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && !reflect.DeepEqual(tt.goval, tt.after) {
				t.Errorf("goSetIndex() got %v expected %v", tt.goval, tt.after)
			}
		})
	}
}
