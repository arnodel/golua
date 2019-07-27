package golib

import (
	"errors"
	"fmt"
	"reflect"

	rt "github.com/arnodel/golua/runtime"
)

// Index tries to find the value of the go value at "index" v. This could mean
// finding a method or a struct field or a map key or a slice index.
func goIndex(t *rt.Thread, u *rt.UserData, key rt.Value) (rt.Value, error) {
	gv := reflect.ValueOf(u.Value())
	meta := u.Metatable()
	field, ok := rt.AsString(key)
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
		return nil, fmt.Errorf("no field or method with name %q", field)
	case reflect.Map:
		goV, err := valueToType(t, key, gv.Type().Key())
		if err != nil {
			return nil, fmt.Errorf("map index or incorrect type: %s", err)
		}
		return reflectToValue(gv.MapIndex(goV), meta), nil
	case reflect.Slice:
		i, ok := rt.ToInt(key)
		if !ok {
			return nil, errors.New("slice index must be an integer")
		}
		return reflectToValue(gv.Index(int(i)), meta), nil
	}
	return nil, errors.New("unable to index")
}

// SetIndex tries to set the value of the index given by key to val.  This could
// mean setting a struct field value or a map value or a slice item.
func goSetIndex(t *rt.Thread, u *rt.UserData, key rt.Value, val rt.Value) error {
	gv := reflect.ValueOf(u.Value())
	switch gv.Kind() {
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
		goVal, err := valueToType(t, val, f.Type())
		if err != nil {
			return fmt.Errorf("struct field of incompatible type: %s", err)
		}
		f.Set(goVal)
		return nil
	case reflect.Map:
		goKey, err := valueToType(t, key, gv.Type().Key())
		if err != nil {
			return fmt.Errorf("map key of incompatible type: %s", err)
		}
		goVal, err := valueToType(t, val, gv.Type().Elem())
		if err != nil {
			return fmt.Errorf("map value set to incompatible type: %s", err)
		}
		gv.SetMapIndex(goKey, goVal)
		return nil
	case reflect.Slice:
		i, ok := rt.ToInt(key)
		if !ok {
			return errors.New("slice idnex must be an integer")
		}
		goVal, err := valueToType(t, val, gv.Type().Elem())
		if err != nil {
			return fmt.Errorf("slice item set to incompatible type: %s", err)
		}
		gv.Index(int(i)).Set(goVal)
		return nil
	}
	return errors.New("unable to set index")
}

// Call tries to call the goValue if it is a function with the given arguments.
func goCall(t *rt.Thread, u *rt.UserData, args []rt.Value) ([]rt.Value, error) {
	gv := reflect.ValueOf(u.Value())
	meta := u.Metatable()
	if gv.Kind() != reflect.Func {
		return nil, fmt.Errorf("%s is not a function", gv.Kind())
	}
	f := gv.Type()
	numParams := f.NumIn()
	goArgs := make([]reflect.Value, numParams)
	isVariadic := f.IsVariadic()
	if isVariadic {
		numParams--
	}
	var goArg reflect.Value
	var err error
	for i := 0; i < numParams; i++ {
		if i < len(args) {
			goArg, err = valueToType(t, args[i], f.In(i))
			if err != nil {
				return nil, err
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
			goArg, err = valueToType(t, args[i+numParams], etcType)
			if err != nil {
				return nil, err
			}
			etc.Index(i).Set(goArg)
		}
		goArgs[numParams] = etc
		goRes = gv.CallSlice(goArgs)
	} else {
		goRes = gv.Call(goArgs)
	}
	res := make([]rt.Value, len(goRes))
	for i, x := range goRes {
		res[i] = reflectToValue(x, meta)
	}
	return res, nil
}

func fillStruct(t *rt.Thread, s reflect.Value, v rt.Value) error {
	var ok bool
	tbl, ok := v.(*rt.Table)
	if !ok {
		return errors.New("fillStruct: can only fill from a table")
	}
	var fk, fv rt.Value
	for {
		fk, fv, ok = tbl.Next(fk)
		if !ok || fk == nil {
			break
		}
		name, ok := fk.(rt.String)
		if !ok {
			return errors.New("fillStruct: table fields must be strings")
		}
		field := s.FieldByName(string(name))
		if field == (reflect.Value{}) {
			return fmt.Errorf("fillStruct: field %q does not exist in struct", name)
		}
		goFv, err := valueToType(t, fv, field.Type())
		if err != nil {
			return err
		}
		field.Set(goFv)
	}
	return nil
}

func valueToFunc(t *rt.Thread, v rt.Value, tp reflect.Type) (reflect.Value, error) {
	fn := func(in []reflect.Value) []reflect.Value {
		args := make([]rt.Value, len(in))
		for i, x := range in {
			args[i] = reflectToValue(x, nil)
		}
		res := make([]rt.Value, tp.NumOut())
		out := make([]reflect.Value, len(res))
		term := rt.NewTermination(res, nil)
		if err := rt.Call(t, v, args, term); err != nil {
			panic(err)
		}
		var err error
		for i, y := range res {
			out[i], err = valueToType(t, y, tp.Out(i))
			if err != nil {
				panic(err)
			}
		}
		return out
	}
	return reflect.MakeFunc(tp, fn), nil
}

func valueToType(t *rt.Thread, v rt.Value, tp reflect.Type) (reflect.Value, error) {
	// Fist we deal with UserData
	if u, ok := v.(*rt.UserData); ok {
		gv := reflect.ValueOf(u.Value())
		if gv.Type().AssignableTo(tp) {
			return gv, nil
		}
		if gv.Type().ConvertibleTo(tp) {
			return gv.Convert(tp), nil
		}
		return reflect.Value{}, fmt.Errorf("%+v is not assignable or convertible to %s", u.Value(), tp.Name())
	}
	switch tp.Kind() {
	case reflect.Ptr:
		if tp.Elem().Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("lua value cannot be converted to %s", tp.Name())
		}
		p := reflect.New(tp.Elem())
		if err := fillStruct(t, p.Elem(), v); err != nil {
			return reflect.Value{}, err
		}
		return p, nil
	case reflect.Struct:
		s := reflect.Zero(tp)
		if err := fillStruct(t, s, v); err != nil {
			return reflect.Value{}, err
		}
		return s, nil
	case reflect.Func:
		return valueToFunc(t, v, tp)
	case reflect.Int:
		x, ok := rt.ToInt(v)
		if ok {
			return reflect.ValueOf(int(x)), nil
		}
	case reflect.Float64:
		x, ok := rt.ToFloat(v)
		if ok {
			return reflect.ValueOf(float64(x)), nil
		}
	case reflect.String:
		x, ok := rt.AsString(v)
		if ok {
			return reflect.ValueOf(string(x)), nil
		}
	case reflect.Bool:
		return reflect.ValueOf(rt.Truth(v)), nil
	case reflect.Interface:
		if reflect.TypeOf(v).Implements(tp) {
			return reflect.ValueOf(v), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("%+v cannot be converted to %s", v, tp.Name())
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
	case reflect.Float32, reflect.Float64:
		return rt.Float(v.Float())
	case reflect.String:
		return rt.String(v.String())
	case reflect.Bool:
		return rt.Bool(v.Bool())
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return rt.String(v.Interface().([]byte))
		}
	}
	return rt.NewUserData(v.Interface(), meta)
}
