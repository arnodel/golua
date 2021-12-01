package runtime

// RuntimeContextDef contains the data necessary to create an new runtime context.
type RuntimeContextDef struct {
	HardLimits     RuntimeResources
	SoftLimits     RuntimeResources
	RequiredFlags  ComplianceFlags
	MessageHandler Callable
}

// RuntimeContext is an interface implemented by Runtime.RuntimeContext().  It
// provides a public interface for the runtime context to be used by e.g.
// libraries (e.g. the runtime package).
type RuntimeContext interface {
	HardLimits() RuntimeResources
	SoftLimits() RuntimeResources
	UsedResources() RuntimeResources

	Status() RuntimeContextStatus
	Parent() RuntimeContext

	RequiredFlags() ComplianceFlags
}

// A ContextTerminationError is an error reserved for when the runtime context
// should be terminated immediately.
type ContextTerminationError struct {
	message string
}

var _ error = ContextTerminationError{}

func (e ContextTerminationError) Error() string {
	return e.message
}

// RuntimeContextStatus describes the status of a context
type RuntimeContextStatus uint16

const (
	StatusLive   RuntimeContextStatus = iota // currently executing
	StatusDone                               // finished successfully (no error)
	StatusError                              // finished with a Lua error
	StatusKilled                             // terminated (either by user or because hard limits were reached)
)

const (
	liveStatusString   = "live"
	doneStatusString   = "done"
	errorStatusString  = "error"
	killedStatusString = "killed"
)

func (s RuntimeContextStatus) String() string {
	switch s {
	case StatusLive:
		return liveStatusString
	case StatusDone:
		return doneStatusString
	case StatusError:
		return errorStatusString
	case StatusKilled:
		return killedStatusString
	default:
		return ""
	}
}

// ComplianceFlags represents constraints that the code running must comply
// with.
type ComplianceFlags uint16

const (
	// Only execute code checks memory availability before allocating memory
	ComplyMemSafe ComplianceFlags = 1 << iota

	// Only execute code that checks cpu availability before executing a
	// computation.
	ComplyCpuSafe

	// Only execute code that complies with IO restrictions (currently only
	// functions that do no IO comply with this)
	ComplyIoSafe
	complyflagsLimit
)

const (
	memSafeString = "memsafe"
	cpuSafeString = "cpusafe"
	ioSafeString  = "iosafe"
)

var complianceFlagNames = map[ComplianceFlags]string{
	ComplyMemSafe: memSafeString,
	ComplyCpuSafe: cpuSafeString,
	ComplyIoSafe:  ioSafeString,
}

var complianceFlagsByName = map[string]ComplianceFlags{
	memSafeString: ComplyMemSafe,
	cpuSafeString: ComplyCpuSafe,
	ioSafeString:  ComplyIoSafe,
}

func (f ComplianceFlags) AddFlagWithName(name string) (ComplianceFlags, bool) {
	fn, ok := complianceFlagsByName[name]
	return fn | f, ok
}

func (f ComplianceFlags) Names() (names []string) {
	var i ComplianceFlags
	for i = 1; i < complyflagsLimit; i <<= 1 {
		if i&f != 0 {
			names = append(names, complianceFlagNames[i])
		}
	}
	return names
}

// RuntimeResources describe amount of resources that code can consume.
// Depending on the context, it could be available resources or consumed
// resources.
type RuntimeResources struct {
	Cpu  uint64
	Mem  uint64
	Time uint64
}

// Remove lowers the resources accounted for in the receiver by the resources
// accounted for in the argument.
func (r RuntimeResources) Remove(r1 RuntimeResources) RuntimeResources {
	if r.Cpu >= r1.Cpu {
		r.Cpu -= r1.Cpu
	}
	if r.Mem >= r1.Mem {
		r.Mem -= r1.Mem
	}
	if r.Time >= r1.Time {
		r.Time -= r1.Time
	}
	return r
}

// Merge treats the receiver and argument as describing resource limits and
// returns the resources describing the intersection of those limits.
func (r RuntimeResources) Merge(r1 RuntimeResources) RuntimeResources {
	if r1.Cpu > 0 && (r.Cpu == 0 || r.Cpu > r1.Cpu) {
		r.Cpu = r1.Cpu
	}
	if r1.Mem > 0 && (r.Mem == 0 || r.Mem > r1.Mem) {
		r.Mem = r1.Mem
	}
	if r1.Time > 0 && (r.Time == 0 || r.Time > r1.Time) {
		r.Time = r1.Time
	}
	return r
}

// Dominates returns r.Merge(r1) == r1, i.e. true iff r1 describes stricter
// limits than r.
func (r RuntimeResources) Dominates(r1 RuntimeResources) bool {
	if r.Cpu > 0 && r1.Cpu >= r.Cpu {
		return false
	}
	if r.Mem > 0 && r1.Mem >= r.Mem {
		return false
	}
	if r.Time > 0 && r1.Time >= r.Time {
		return false
	}
	return true
}
