package runtimelib

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/safeio"
)

var LibLoader = packagelib.Loader{
	Load: load,
	Name: "runtime",
}

func load(r *rt.Runtime) (rt.Value, func()) {
	if !rt.QuotasAvailable {
		return rt.NilValue, nil
	}
	pkg := rt.NewTable()

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "callcontext", callcontext, 2, true),
		r.SetEnvGoFunc(pkg, "context", context, 0, false),
		r.SetEnvGoFunc(pkg, "killcontext", killnow, 1, false),
		r.SetEnvGoFunc(pkg, "stopcontext", stopnow, 1, false),
		r.SetEnvGoFunc(pkg, "contextdue", due, 1, false),
	)

	createContextMetatable(r)

	return rt.TableValue(pkg), nil
}

func context(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	ctx := newContextValue(t.Runtime, t.RuntimeContext())
	return c.PushingNext1(t.Runtime, ctx), nil
}

func callcontext(t *rt.Thread, c *rt.GoCont) (next rt.Cont, retErr error) {
	quotas, err := c.TableArg(0)
	if err != nil {
		return nil, err
	}
	var (
		flagsV      = quotas.Get(rt.StringValue("flags"))
		limitsV     = quotas.Get(rt.StringValue("kill"))
		softLimitsV = quotas.Get(rt.StringValue("stop"))
		fsV         = quotas.Get(rt.StringValue("fs"))
		hardLimits  rt.RuntimeResources
		softLimits  rt.RuntimeResources
		f           = c.Arg(1)
		fArgs       = c.Etc()
		flags       rt.ComplianceFlags
	)
	if !limitsV.IsNil() {
		var err error
		hardLimits, err = getResources(t, limitsV)
		if err != nil {
			return nil, err
		}
	}
	if !softLimitsV.IsNil() {
		var err error
		softLimits, err = getResources(t, softLimitsV)
		if err != nil {
			return nil, err
		}
	}
	if !flagsV.IsNil() {
		flagsStr, ok := flagsV.TryString()
		if !ok {
			return nil, errors.New("flags must be a string")
		}
		for _, name := range strings.Fields(flagsStr) {
			flags, ok = flags.AddFlagWithName(name)
			if !ok {
				return nil, fmt.Errorf("unknown flag: %q", name)
			}
		}
	}
	fsRule, err := getFSAccessRuleset(t, fsV)
	if err != nil {
		return nil, err
	}
	next = c.Next()
	res := rt.NewTerminationWith(c, 0, true)

	ctx, err := t.CallContext(rt.RuntimeContextDef{
		HardLimits:    hardLimits,
		SoftLimits:    softLimits,
		RequiredFlags: flags,
		FSAccessRule:  fsRule,
	}, func() error {
		return rt.Call(t, f, fArgs, res)
	})
	t.Push1(next, newContextValue(t.Runtime, ctx))
	switch ctx.Status() {
	case rt.StatusDone:
		t.Push(next, res.Etc()...)
	case rt.StatusError:
		t.Push1(next, rt.ErrorValue(err))
	}
	return next, nil
}

func getResources(t *rt.Thread, resources rt.Value) (res rt.RuntimeResources, err error) {
	res.Cpu, err = getResVal(t, resources, cpuString)
	if err != nil {
		return
	}
	res.Memory, err = getResVal(t, resources, memoryString)
	if err != nil {
		return
	}
	res.Millis, err = getTimeVal(t, resources)
	if err != nil {
		return
	}
	return
}

func getResVal(t *rt.Thread, resources rt.Value, key rt.Value) (uint64, error) {
	val, err := rt.Index(t, resources, key)
	if err != nil {
		return 0, err
	}
	return validateResVal(key, val)
}

func validateResVal(key rt.Value, val rt.Value) (uint64, error) {
	if val.IsNil() {
		return 0, nil
	}
	n, ok := rt.ToIntNoString(val)
	if !ok {
		name, _ := key.ToString()
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	if n <= 0 {
		name, _ := key.ToString()
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return uint64(n), nil
}

func getTimeVal(t *rt.Thread, resources rt.Value) (uint64, error) {
	val, err := rt.Index(t, resources, secondsString)
	if err != nil {
		return 0, err
	}
	if !val.IsNil() {
		return validateTimeVal(val, 1000, secondsName)
	}
	val, err = rt.Index(t, resources, millisString)
	if err != nil {
		return 0, err
	}
	return validateTimeVal(val, 1, millisName)
}

func validateTimeVal(val rt.Value, factor float64, name string) (uint64, error) {
	if val.IsNil() {
		return 0, nil
	}
	s, ok := rt.ToFloat(val)
	if !ok {
		return 0, fmt.Errorf("%s must be a numeric value", name)
	}
	if s <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}
	return uint64(s * factor), nil
}

func getFSAccessRuleset(t *rt.Thread, val rt.Value) (safeio.FSAccessRule, error) {
	if val.IsNil() {
		return nil, nil
	}
	sz, err := rt.IntLen(t, val)
	if err != nil {
		return nil, err
	}
	if sz < 1 {
		return nil, errors.New("fs ruleset must be non-empty list of rules")
	}
	var rules []safeio.FSAccessRule
	for i := int64(1); i <= sz; i++ {
		ruleV, err := rt.Index(t, val, rt.IntValue(i))
		if err != nil {
			return nil, err
		}
		rule, err := getFSAccessRule(ruleV)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return safeio.MergeFSAccessRules(rules...), nil
}

func getFSAccessRule(val rt.Value) (safeio.FSAccessRule, error) {
	tbl, ok := val.TryTable()
	if !ok {
		return nil, errors.New("fs rules must be in the form of tables")
	}
	allowed, err := fsActionsFromValue(tbl.Get(allowedString))
	if err != nil {
		return nil, fmt.Errorf("error in %s value: %w", allowedName, err)
	}
	denied, err := fsActionsFromValue(tbl.Get(deniedString))
	if err != nil {
		return nil, fmt.Errorf("error in %s value: %w", deniedName, err)
	}
	var prefix string
	if prefixV := tbl.Get(prefixString); !prefixV.IsNil() {
		prefix, ok = prefixV.TryString()
		if !ok {
			return nil, fmt.Errorf("%s value in fs rule must be a string", prefixName)
		}
		prefix = filepath.Clean(prefix)
	}
	return safeio.PrefixFSAccessRule{
		Prefix:         prefix,
		AllowedActions: allowed,
		DeniedActions:  denied,
	}, nil
}

func fsActionsFromValue(val rt.Value) (safeio.FSAction, error) {
	if val.IsNil() {
		return 0, nil
	}
	s, ok := val.TryString()
	if !ok {
		return 0, errors.New("fs actions must be strings")
	}
	var actions safeio.FSAction
	for _, c := range s {
		switch c {
		case 'r':
			actions |= safeio.ReadFileAction
		case 'w':
			actions |= safeio.WriteFileAction
		case 'c':
			actions |= safeio.CreateFileAction
		case 'd':
			actions |= safeio.DeleteFileAction
		case 'C':
			actions |= safeio.CreateFileInDirAction
		case '*':
			actions |= safeio.AllFileActions
		default:
			return 0, fmt.Errorf("invalid file action '%c' (expect rwcdV*)", c)
		}
	}
	return actions, nil
}

const (
	secondsName = "seconds"
	millisName  = "millis"
	cpuName     = "cpu"
	memoryName  = "memory"

	allowedName = "allowed"
	deniedName  = "denied"
	prefixName  = "prefix"
)

var (
	secondsString = rt.StringValue(secondsName)
	millisString  = rt.StringValue(millisName)
	cpuString     = rt.StringValue(cpuName)
	memoryString  = rt.StringValue(memoryName)

	allowedString = rt.StringValue(allowedName)
	deniedString  = rt.StringValue(deniedName)
	prefixString  = rt.StringValue(prefixName)
)
