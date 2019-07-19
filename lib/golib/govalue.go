package golib

import (
	"reflect"

	rt "github.com/arnodel/golua/runtime"
)

type GoValue struct {
	value reflect.Value
}

func ToGoValue(x interface{}) GoValue {
	return GoValue{value: reflect.ValueOf(x)}
}

func (g GoValue) Index(v rt.Value, meta *rt.Table) (rt.Value, bool) {
	switch g.value.Kind() {
	case reflect.Struct:
		field, ok := rt.AsString(v)
		if !ok {
			return nil, false
		}
		m := g.value.MethodByName(string(field))
		if m != (reflect.Value{}) {
			return reflectToValue(m, meta), true
		}
		f := g.value.FieldByName(string(field))
		if f != (reflect.Value{}) {
			return reflectToValue(f, meta), true
		}
		return nil, false
	case reflect.Interface:
		field, ok := rt.AsString(v)
		if !ok {
			return nil, false
		}
		m := g.value.MethodByName(string(field))
		if m != (reflect.Value{}) {
			return reflectToValue(m, meta), true
		}
		return nil, false
	case reflect.Map:
		goV := valueToType(v, g.value.Type().Key())
		if goV == (reflect.Value{}) {
			return nil, false
		}
		return reflectToValue(g.value.MapIndex(goV), meta), true
	case reflect.Slice:
		i, ok := rt.ToInt(v)
		if !ok {
			return nil, false
		}
		return reflectToValue(g.value.Index(int(i)), meta), true
	}
	return nil, false
}

func (g GoValue) SetIndex(key rt.Value, val rt.Value) bool {
	switch g.value.Kind() {
	case reflect.Struct:
		field, ok := rt.AsString(key)
		if !ok {
			return false
		}
		f := g.value.FieldByName(string(field))
		if f == (reflect.Value{}) {
			return false
		}
		goVal := valueToType(val, g.value.Type())
		if goVal == (reflect.Value{}) {
			return false
		}
		g.value.Set(goVal)
		return true
	case reflect.Map:
		goKey := valueToType(key, g.value.Type().Key())
		if goKey == (reflect.Value{}) {
			return false
		}
		goVal := valueToType(val, g.value.Type().Elem())
		if goVal == (reflect.Value{}) {
			return false
		}
		g.value.SetMapIndex(goKey, goVal)
		return true
	case reflect.Slice:
		i, ok := rt.ToInt(key)
		if !ok {
			return false
		}
		goVal := valueToType(val, g.value.Type().Elem())
		if goVal == (reflect.Value{}) {
			return false
		}
		g.value.Index(int(i)).Set(goVal)
		return true
	}
	return false
}

func (g GoValue) Call(args []rt.Value, meta *rt.Table) ([]rt.Value, bool) {
	if g.value.Kind() != reflect.Func {
		return nil, false
	}
	f := g.value.Type()
	if f.NumIn() != len(args) {
		return nil, false
	}
	goArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		goArg := valueToType(arg, f.In(i))
		if goArg == (reflect.Value{}) {
			return nil, false
		}
		goArgs[i] = goArg
	}
	goRes := g.value.Call(goArgs)
	res := make([]rt.Value, len(goRes))
	for i, x := range goRes {
		res[i] = reflectToValue(x, meta)
	}
	return res, true
}

func valueToType(v rt.Value, tp reflect.Type) reflect.Value {
	if goVal, ok := v.(GoValue); ok {
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
