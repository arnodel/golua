package golib_test

import (
	"fmt"
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/lib/golib"
	"github.com/arnodel/golua/luatesting"
	rt "github.com/arnodel/golua/runtime"
)

type TestStruct struct {
	Age  int
	Name string
}

func (t TestStruct) Descr() string {
	return fmt.Sprintf("age: %d, name: %s", t.Age, t.Name)
}

func (t *TestStruct) Mix(u *TestStruct) *TestStruct {
	return &TestStruct{
		Age:  t.Age + u.Age,
		Name: t.Name + "-" + u.Name,
	}
}

func twice(f func(int) int) func(int) int {
	return func(n int) int {
		return f(f(n))
	}
}

func setup(r *rt.Runtime) func() {
	cleanup := lib.LoadAll(r)
	g := r.GlobalEnv()
	r.SetEnv(g, "hello", rt.StringValue("world"))
	r.SetEnv(g, "double", golib.NewGoValue(r, func(x int) int { return 2 * x }))
	r.SetEnv(g, "polly", golib.NewGoValue(r, TestStruct{Age: 10, Name: "Polly"}))
	r.SetEnv(g, "ben", golib.NewGoValue(r, &TestStruct{Age: 5, Name: "Ben"}))
	r.SetEnv(g, "mapping", golib.NewGoValue(r, map[string]int{"answer": 42}))
	r.SetEnv(g, "slice", golib.NewGoValue(r, []string{"I", "am", "here"}))
	r.SetEnv(g, "sprintf", golib.NewGoValue(r, fmt.Sprintf))
	r.SetEnv(g, "twice", golib.NewGoValue(r, twice))
	r.SetEnv(g, "panic", golib.NewGoValue(r, func() { panic("OMG") }))
	return cleanup
}

func TestGoLib(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", setup)
}
