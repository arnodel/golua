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

// SetIndex tries to set the value of the index given by key to val.  This could
// mean setting a struct field value or a map value or a slice item.
func (g GoValue) SetIndex(key rt.Value, val rt.Value) error {
	gv := g.value
	switch g.value.Kind() {
	case reflect.Ptr:
		gv = gv.Elem()
		if gv.Kind() != reflect.Struct {
			return errors.New("can only set pointer index when pointing to a struct")
		}
		fallthrough
	case reflect.Struct:
		field, ok := rt.AsString(key)
		if !ok {
			return errors.New("can only set struct index for a string")
		}
		f := gv.FieldByName(string(field))
		if f == (reflect.Value{}) {
			return errors.New("struct does not have field: " + string(field))
		}
		if !f.CanSet() {
			return errors.New("struct field cannot be set")
		}
		goVal := valueToType(val, f.Type())
		if goVal == (reflect.Value{}) {
			return errors.New("struct field set to incompatible type")
		}
		f.Set(goVal)
		return nil
	case reflect.Map:
		goKey := valueToType(key, gv.Type().Key())
		if goKey == (reflect.Value{}) {
			return errors.New("map key of incompatible type")
		}
		goVal := valueToType(val, gv.Type().Elem())
		if goVal == (reflect.Value{}) {
			return errors.New("map value set to incompatible type")
		}
		gv.SetMapIndex(goKey, goVal)
		return nil
	case reflect.Slice:
		i, ok := rt.ToInt(key)
		if !ok {
			return errors.New("slice idnex must be an integer")
		}
		goVal := valueToType(val, gv.Type().Elem())
		if goVal == (reflect.Value{}) {
			return errors.New("slice item set to incompatible type")
		}
		gv.Index(int(i)).Set(goVal)
		return nil
	}
	return errors.New("unable to set index")
}

// Call tries to call the goValue if it is a function with the given arguments.
func (g GoValue) Call(args []rt.Value, meta *rt.Table) ([]rt.Value, error) {
	if g.value.Kind() != reflect.Func {
		return nil, errors.New("not a function")
	}
	f := g.value.Type()
	numParams := f.NumIn()
	goArgs := make([]reflect.Value, numParams)
	isVariadic := f.IsVariadic()
	if isVariadic {
		numParams--
	}
	var goArg reflect.Value
	for i := 0; i < numParams; i++ {
		if i < len(args) {
			goArg = valueToType(args[i], f.In(i))
			if goArg == (reflect.Value{}) {
				return nil, errors.New("argument of incorrect type")
			}
		} else {
			goArg = reflect.Zero(f.In(i))
		}
		goArgs[i] = goArg
	}
	var goRes []reflect.Value
	if isVariadic {
		etcSliceType := f.In(numParams)
		etcType := etcSliceType.Elem()
		etcLen := len(args) - numParams
		etc := reflect.MakeSlice(etcSliceType, etcLen, etcLen)
		for i := 0; i < etcLen; i++ {
			goArg = valueToType(args[i+numParams], etcType)
			if goArg == (reflect.Value{}) {
				return nil, errors.New("argument of incorrect type")
			}
			etc.Index(i).Set(goArg)
		}
		goArgs[numParams] = etc
		goRes = g.value.CallSlice(goArgs)
	} else {
		goRes = g.value.Call(goArgs)
	}
	res := make([]rt.Value, len(goRes))
	for i, x := range goRes {
		res[i] = reflectToValue(x, meta)
	}
	return res, nil
}

func fillStruct(s reflect.Value, v rt.Value) bool {
	var ok bool
	t, ok := v.(*rt.Table)
	if !ok {
		return false
	}
	var fk, fv rt.Value
	for {
		fk, fv, ok = t.Next(fk)
		if !ok || fk == nil {
			break
		}
		name, ok := fk.(rt.String)
		if !ok {
			return false
		}
		field := s.FieldByName(string(name))
		if field == (reflect.Value{}) {
			return false
		}
		goFv := valueToType(fv, field.Type())
		if goFv == (reflect.Value{}) {
			return false
		}
		field.Set(goFv)
	}
	return true
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
	case reflect.Ptr:
		if tp.Elem().Kind() != reflect.Struct {
			return reflect.Value{}
		}
		p := reflect.New(tp.Elem())
		if !fillStruct(p.Elem(), v) {
			return reflect.Value{}
		}
		return p
	case reflect.Struct:
		s := reflect.Zero(tp)
		if !fillStruct(s, v) {
			return reflect.Value{}
		}
		return s
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
	case reflect.Interface:
		ok = reflect.TypeOf(v).Implements(tp)
		if ok {
			goV = v
		}
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
