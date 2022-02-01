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
	field, ok := key.ToString()
	if ok {
		// First try a method
		m := gv.MethodByName(string(field))
		if m != (reflect.Value{}) {
			return reflectToValue(m, meta), nil
		}
		if gv.CanAddr() {
			// Is that even possible?
			m = gv.Addr().MethodByName(string(field))
			if m != (reflect.Value{}) {
				return reflectToValue(m, meta), nil
			}
		}
	}
	switch gv.Kind() {
	case reflect.Ptr:
		gv = gv.Elem()
		if gv.Kind() != reflect.Struct {
			return rt.NilValue, errors.New("can only index a pointer when to a struct")
		}
		fallthrough
	case reflect.Struct:
		if !ok {
			return rt.NilValue, errors.New("can only index a struct with a string")
		}
		f := gv.FieldByName(string(field))
		if f != (reflect.Value{}) {
			return reflectToValue(f, meta), nil
		}
		return rt.NilValue, fmt.Errorf("no field or method with name %q", field)
	case reflect.Map:
		goV, err := valueToType(t, key, gv.Type().Key())
		if err != nil {
			return rt.NilValue, fmt.Errorf("map index or incorrect type: %s", err)
		}
		return reflectToValue(gv.MapIndex(goV), meta), nil
	case reflect.Slice:
		i, ok := rt.ToInt(key)
		if !ok {
			return rt.NilValue, errors.New("slice index must be an integer")
		}
		if i < 0 || int(i) >= gv.Len() {
			return rt.NilValue, errors.New("index out of slice bounds")
		}
		return reflectToValue(gv.Index(int(i)), meta), nil
	}
	return rt.NilValue, errors.New("unable to index")
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
		field, ok := key.ToString()
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
			return errors.New("slice index must be an integer")
		}
		if i < 0 || int(i) >= gv.Len() {
			return errors.New("slice index out of bounds")
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
func goCall(t *rt.Thread, u *rt.UserData, args []rt.Value) (res []rt.Value, err error) {
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
	for i := 0; i < numParams; i++ {
		if i < len(args) {
			goArg, err = valueToType(t, args[i], f.In(i))
			if err != nil {
				return
			}
		} else {
			goArg = reflect.Zero(f.In(i))
		}
		goArgs[i] = goArg
	}
	var goRes []reflect.Value
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in go call: %v", r)
		}
	}()
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
	res = make([]rt.Value, len(goRes))
	for i, x := range goRes {
		res[i] = reflectToValue(x, meta)
	}
	return
}

func fillStruct(t *rt.Thread, s reflect.Value, v rt.Value) error {
	var ok bool
	tbl, ok := v.TryTable()
	if !ok {
		return errors.New("fillStruct: can only fill from a table")
	}
	var fk, fv rt.Value
	for {
		fk, fv, ok = tbl.Next(fk)
		if !ok || fk.IsNil() {
			break
		}
		name, ok := fk.TryString()
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
	meta := getMeta(t.Runtime)
	fn := func(in []reflect.Value) []reflect.Value {
		args := make([]rt.Value, len(in))
		for i, x := range in {
			args[i] = reflectToValue(x, meta)
		}
		res := make([]rt.Value, tp.NumOut())
		out := make([]reflect.Value, len(res))
		term := rt.NewTermination(t.CurrentCont(), res, nil)
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

var runtimeValueType = reflect.TypeOf(rt.Value{})

func valueToType(t *rt.Thread, v rt.Value, tp reflect.Type) (reflect.Value, error) {
	if tp == runtimeValueType {
		return reflect.ValueOf(v), nil
	}
	// Fist we deal with UserData
	if u, ok := v.TryUserData(); ok {
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
		x, ok := v.ToString()
		if ok {
			return reflect.ValueOf(string(x)), nil
		}
	case reflect.Bool:
		return reflect.ValueOf(rt.Truth(v)), nil
	case reflect.Slice:
		if tp.Elem().Kind() == reflect.Uint8 {
			s, ok := v.TryString()
			if ok {
				return reflect.ValueOf([]byte(s)), nil
			}
		}
	case reflect.Interface:
		iface := v.Interface()
		if reflect.TypeOf(iface).Implements(tp) {
			return reflect.ValueOf(iface), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("%+v cannot be converted to %s", v, tp.Name())
}

func reflectToValue(v reflect.Value, meta *rt.Table) rt.Value {
	if v == (reflect.Value{}) {
		return rt.NilValue
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rt.IntValue(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rt.IntValue(int64(v.Uint()))
	case reflect.Float32, reflect.Float64:
		return rt.FloatValue(v.Float())
	case reflect.String:
		return rt.StringValue(v.String())
	case reflect.Bool:
		return rt.BoolValue(v.Bool())
	case reflect.Slice:
		if v.IsNil() {
			return rt.NilValue
		}
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return rt.StringValue(string(v.Interface().([]byte)))
		}
	case reflect.Ptr:
		if v.IsNil() {
			return rt.NilValue
		}
		switch x := v.Interface().(type) {
		case *rt.Table:
			return rt.TableValue(x)
		case *rt.UserData:
			return rt.UserDataValue(x)
		}
	case reflect.Interface:
		if v.IsNil() {
			return rt.NilValue
		}
	}
	return rt.UserDataValue(rt.NewUserData(v.Interface(), meta))
}
