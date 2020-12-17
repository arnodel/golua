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
	testTable.Set(rt.StringValue("key"), rt.IntValue(123))
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
			want: rt.NilValue,
		},
		{
			name: "int",
			arg:  int(8),
			want: rt.IntValue(8),
		},
		{
			name: "int8",
			arg:  int8(-33),
			want: rt.IntValue(-33),
		},
		{
			name: "uint",
			arg:  uint(21),
			want: rt.IntValue(21),
		},
		{
			name: "uint16",
			arg:  uint16(777),
			want: rt.IntValue(777),
		},
		{
			name: "float64",
			arg:  float64(1.2),
			want: rt.FloatValue(1.2),
		},
		{
			name: "string",
			arg:  string("hello"),
			want: rt.StringValue("hello"),
		},
		{
			name: "bool",
			arg:  true,
			want: rt.BoolValue(true),
		},
		{
			name: "[]byte",
			arg:  []byte("bonjour"),
			want: rt.StringValue("bonjour"),
		},
		{
			name: "lua table",
			arg:  testTable,
			want: rt.TableValue(testTable),
		},
		{
			name: "lua userdata",
			arg:  testUdata,
			want: rt.UserDataValue(testUdata),
		},
		{
			name: "non luable type",
			arg:  testStruct,
			want: rt.UserDataValue(rt.NewUserData(testStruct, meta)),
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
		v       interface{}
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
			v:    int64(123),
			want: int(123),
		},
		{
			name: "integral rt.Float to int",
			v:    float64(111),
			want: int(111),
		},
		{
			name: "integral string to int",
			v:    "432",
			want: int(432),
		},
		{
			name: "rt.Float to float64",
			v:    float64(1.3),
			want: float64(1.3),
		},
		{
			name: "rt.Int to float64",
			v:    int64(-123),
			want: float64(-123),
		},
		{
			name: "floaty string to float64",
			v:    "3.14",
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
			name: "int64(0) to bool",
			v:    int64(0),
			want: true,
		},
		{
			name: "false to bool",
			v:    false,
			want: false,
		},
		{
			name: "true to bool",
			v:    true,
			want: true,
		},
		{
			name: "empty string to bool",
			v:    "",
			want: true,
		},
		{
			name: "rt.String to []byte",
			v:    "foo",
			want: []byte("foo"),
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueToType(thread, rt.AsValue(tt.v), reflect.TypeOf(tt.want))
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
		v       interface{}
		wantErr bool
	}{
		{
			name:    "not a table",
			before:  reflect.ValueOf(struct{}{}),
			v:       int64(12),
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
			err := fillStruct(thread, tt.before, rt.AsValue(tt.v))
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
		key             interface{}
		want            interface{}
		wantErr         bool
		doNotCheckValue bool
	}{
		{
			name:            "method on struct pointer",
			goval:           testErr,
			key:             "Error",
			doNotCheckValue: true,
		},
		{
			name:    "pointer to non struct",
			goval:   &testInt,
			key:     "x",
			wantErr: true,
		},
		{
			name:    "non-string index for struct",
			goval:   struct{ Foo int }{},
			key:     true,
			wantErr: true,
		},
		{
			name:    "index for struct not referring to a field",
			goval:   struct{ Foo int }{},
			key:     "Bar",
			wantErr: true,
		},
		{
			name:  "index for struct referring to a field",
			goval: struct{ Foo int }{Foo: 12},
			key:   "Foo",
			want:  int64(12),
		},
		{
			name:    "map index of incompatible type",
			goval:   map[int]int{},
			key:     "hi",
			wantErr: true,
		},
		{
			name:  "map index of compatible type",
			goval: map[string]int{"hi": 34},
			key:   "hi",
			want:  int64(34),
		},
		{
			name:    "non-integral slice index",
			goval:   []string{"hi", "there"},
			key:     "bad",
			wantErr: true,
		},
		{
			name:  "integral slice index within bounds",
			goval: []string{"hi", "there"},
			key:   float64(1),
			want:  "there",
		},
		{
			name:    "integral slice index greater than length-1",
			goval:   []string{"hi", "there"},
			key:     float64(2),
			wantErr: true,
		},
		{
			name:    "integral slice index negative",
			goval:   []string{"hi", "there"},
			key:     int64(-1),
			wantErr: true,
		},
		{
			name:    "unsupported type (function)",
			goval:   func() {},
			key:     int64(1),
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := goIndex(thread, rt.NewUserData(tt.goval, meta), rt.AsValue(tt.key))
			if (err != nil) != tt.wantErr {
				t.Errorf("goIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !tt.doNotCheckValue && !reflect.DeepEqual(got, rt.AsValue(tt.want)) {
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
		key     interface{} // Will be converted with rt.AsValue
		val     interface{} // Will be converted with rt.AsValue
		wantErr bool
	}{
		{
			name:    "pointer to non struct",
			goval:   &testInt,
			key:     "key",
			val:     "val",
			wantErr: true,
		},
		{
			name:    "non string struct index",
			goval:   &struct{}{},
			key:     true,
			val:     int64(10),
			wantErr: true,
		},
		{
			name:    "non existing struct field",
			goval:   &struct{ Foo int }{},
			key:     "Bar",
			val:     int64(1),
			wantErr: true,
		},
		{
			name:    "struct field set to incompatible type",
			goval:   &struct{ Foo int }{},
			key:     "Foo",
			val:     "hi",
			wantErr: true,
		},
		{
			name:  "struct field set to incompatible type",
			goval: &struct{ Foo int }{},
			key:   "Foo",
			val:   int64(12),
			after: &struct{ Foo int }{Foo: 12},
		},
		{
			name:    "struct field non settable",
			goval:   struct{ Foo int }{},
			key:     "Foo",
			val:     int64(12),
			wantErr: true,
		},
		{
			name:    "map key of incompatible type",
			goval:   map[int]string{},
			key:     "three",
			val:     int64(444),
			wantErr: true,
		},
		{
			name:    "map value of incompatible type",
			goval:   map[int]string{},
			key:     int64(444),
			val:     false,
			wantErr: true,
		},
		{
			name:  "map success",
			goval: map[int]string{},
			key:   int64(444),
			val:   "chouette",
			after: map[int]string{444: "chouette"},
		},
		{
			name:    "non integer slice index",
			goval:   []int{3, 2, 1},
			key:     "deux",
			val:     int64(12),
			wantErr: true,
		},
		{
			name:    "negative slice index",
			goval:   []int{3, 2, 1},
			key:     int64(-1),
			val:     int64(12),
			wantErr: true,
		},
		{
			name:    "slice index > len-1",
			goval:   []int{3, 2, 1},
			key:     int64(3),
			val:     int64(12),
			wantErr: true,
		},
		{
			name:    "slice value of incompatible type",
			goval:   []int{3, 2, 1},
			key:     int64(2),
			val:     true,
			wantErr: true,
		},
		{
			name:  "successful slice",
			goval: []int{3, 2, 1},
			key:   int64(1),
			val:   int64(12),
			after: []int{3, 12, 1},
		},
		{
			name:    "unsupported go type",
			goval:   false,
			key:     int64(1),
			val:     int64(2),
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := rt.NewUserData(tt.goval, meta)
			err := goSetIndex(thread, u, rt.AsValue(tt.key), rt.AsValue(tt.val))
			if (err != nil) != tt.wantErr {
				t.Errorf("goSetIndex() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && !reflect.DeepEqual(tt.goval, tt.after) {
				t.Errorf("goSetIndex() got %v expected %v", tt.goval, tt.after)
			}
		})
	}
}
