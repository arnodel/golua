package golib

import (
	"errors"
	"reflect"

	rt "github.com/arnodel/golua/runtime"
)

// A GoValue holds any go value.
type GoValue struct {
	value reflect.Value
}

// ToGoValue turn any go value into a GoValue
func ToGoValue(x interface{}) GoValue {
	return GoValue{value: reflect.ValueOf(x)}
}

// Index tries to find the value of the go value at "index" v. This could mean
// finding a method or a struct field or a map key or a slice index.
func (g GoValue) Index(v rt.Value, meta *rt.Table) (rt.Value, error) {
	gv := g.value
	field, ok := rt.AsString(v)
	if ok {
		// First try a method
		m := gv.MethodByName(string(field))
		if m != (reflect.Value{}) {
			return reflectToValue(m, meta), nil
		}
	}
	switch gv.Kind() {
	case reflect.Ptr:
		gv = gv.Elem()
		if gv.Kind() != reflect.Struct {
			return nil, errors.New("can only index a pointer when to a struct")
		}
		fallthrough
	case reflect.Struct:
		if !ok {
			return nil, errors.New("can only index a struct with a string")
		}
		f := gv.FieldByName(string(field))
		if f != (reflect.Value{}) {
			return reflectToValue(f, meta), nil
		}
		return nil, errors.New("could not find field or method with name: " + string(field))
	case reflect.Map:
		goV := valueToType(v, gv.Type().Key())
		if goV == (reflect.Value{}) {
			return nil, errors.New("map index or incorrect type")
		}
		return reflectToValue(gv.MapIndex(goV), meta), nil
	case reflect.Slice:
		i, ok := rt.ToInt(v)
		if !ok {
			return nil, errors.New("slice index must be an integer")
		}
		return reflectToValue(gv.Index(int(i)), meta), nil
	}
	return nil, errors.New("unable to index")
}

func (g GoValue) SetIndex(key rt.Value, val rt.Value) bool {
	gv := g.value
	switch g.value.Kind() {
	case reflect.Ptr:
		gv = gv.Elem()
		if gv.Kind() != reflect.Struct {
			return false
		}
		fallthrough
	case reflect.Struct:
		field, ok := rt.AsString(key)
		if !ok {
			return false
		}
		f := gv.FieldByName(string(field))
		if f == (reflect.Value{}) {
			return false
		}
		if !f.CanSet() {
			return false
		}
		goVal := valueToType(val, f.Type())
		if goVal == (reflect.Value{}) {
			return false
		}
		f.Set(goVal)
		return true
	case reflect.Map:
		goKey := valueToType(key, gv.Type().Key())
		if goKey == (reflect.Value{}) {
			return false
		}
		goVal := valueToType(val, gv.Type().Elem())
		if goVal == (reflect.Value{}) {
			return false
		}
		gv.SetMapIndex(goKey, goVal)
		return true
	case reflect.Slice:
		i, ok := rt.ToInt(key)
		if !ok {
			return false
		}
		goVal := valueToType(val, gv.Type().Elem())
		if goVal == (reflect.Value{}) {
			return false
		}
		gv.Index(int(i)).Set(goVal)
		return true
	}
	return false
}

func (g GoValue) Call(args []rt.Value, meta *rt.Table) ([]rt.Value, error) {
	if g.value.Kind() != reflect.Func {
		return nil, errors.New("not a function")
	}
	f := g.value.Type()
	if f.NumIn() != len(args) {
		return nil, errors.New("wrong number of arguments")
	}
	goArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		goArg := valueToType(arg, f.In(i))
		if goArg == (reflect.Value{}) {
			return nil, errors.New("argument of incorrect type")
		}
		goArgs[i] = goArg
	}
	goRes := g.value.Call(goArgs)
	res := make([]rt.Value, len(goRes))
	for i, x := range goRes {
		res[i] = reflectToValue(x, meta)
	}
	return res, nil
}

func valueToType(v rt.Value, tp reflect.Type) reflect.Value {
	if goVal, _, ok := ValueToGoValue(v); ok {
		if goVal.value.Type().AssignableTo(tp) {
			return goVal.value
		}
		return reflect.Value{}
	}
	var goV interface{}
	var ok bool
	switch tp.Kind() {
	case reflect.Int:
		var x rt.Int
		x, ok = rt.ToInt(v)
		goV = int(x)
	case reflect.Float64:
		var x rt.Float
		x, ok = rt.ToFloat(v)
		goV = float64(x)
	case reflect.String:
		var x rt.String
		x, ok = rt.AsString(v)
		goV = string(x)
	case reflect.Bool:
		goV = rt.Truth(v)
		ok = true
	}
	if !ok {
		return reflect.Value{}
	}
	return reflect.ValueOf(goV)
}

func reflectToValue(v reflect.Value, meta *rt.Table) rt.Value {
	if v == (reflect.Value{}) {
		return nil
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rt.Int(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rt.Int(v.Uint())
	case reflect.String:
		return rt.String(v.String())
	case reflect.Bool:
		return rt.Bool(v.Bool())
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return rt.String(v.Interface().([]byte))
		}
	}
	return rt.NewUserData(GoValue{value: v}, meta)
}
