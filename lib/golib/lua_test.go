package golib_test

import (
	"fmt"
	"testing"

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

func setup(r *rt.Runtime) {
	g := r.GlobalEnv()
	rt.SetEnv(g, "hello", rt.String("world"))
	rt.SetEnv(g, "double", golib.NewGoValue(r, func(x int) int { return 2 * x }))
	rt.SetEnv(g, "polly", golib.NewGoValue(r, TestStruct{Age: 10, Name: "Polly"}))
	rt.SetEnv(g, "ben", golib.NewGoValue(r, &TestStruct{Age: 5, Name: "Ben"}))
	rt.SetEnv(g, "mapping", golib.NewGoValue(r, map[string]int{"answer": 42}))
	rt.SetEnv(g, "slice", golib.NewGoValue(r, []string{"I", "am", "here"}))
	rt.SetEnv(g, "sprintf", golib.NewGoValue(r, fmt.Sprintf))
	rt.SetEnv(g, "twice", golib.NewGoValue(r, twice))
}

func TestGoLib(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", setup)
}
