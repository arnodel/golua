package runtime

// RuntimeContextDef contains the data necessary to create an new runtime context.
type RuntimeContextDef struct {
	HardLimits     RuntimeResources
	SoftLimits     RuntimeResources
	RequiredFlags  ComplianceFlags
	MessageHandler Callable
	GCPolicy
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

	SetStopLevel(StopLevel)
	Due() bool

	GCPolicy() GCPolicy
}

type StopLevel uint8

const (
	SoftStop StopLevel = 1 << iota // Forces the context to be due
	HardStop                       // Forces the context to terminate
)

// A ContextTerminationError is an error reserved for when the runtime context
// should be terminated immediately.
type ContextTerminationError struct {
	message string
}

var _ error = ContextTerminationError{}

// Error string for a ContextTerminationError
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

	// Only execute code that is time safe (i.e. it will not block on long
	// running ops, typically IO)
	ComplyTimeSafe

	complyflagsLimit
)

const (
	memSafeString  = "memsafe"
	cpuSafeString  = "cpusafe"
	timeSafeString = "timesafe"
	ioSafeString   = "iosafe"
)

var complianceFlagNames = map[ComplianceFlags]string{
	ComplyMemSafe:  memSafeString,
	ComplyCpuSafe:  cpuSafeString,
	ComplyTimeSafe: timeSafeString,
	ComplyIoSafe:   ioSafeString,
}

var complianceFlagsByName = map[string]ComplianceFlags{
	memSafeString:  ComplyMemSafe,
	cpuSafeString:  ComplyCpuSafe,
	timeSafeString: ComplyTimeSafe,
	ioSafeString:   ComplyIoSafe,
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
// resources.  For available resources, 0 means unlimited.
type RuntimeResources struct {
	Cpu    uint64
	Memory uint64
	Millis uint64
}

// Remove lowers the resources accounted for in the receiver by the resources
// accounted for in the argument.
func (r RuntimeResources) Remove(v RuntimeResources) RuntimeResources {
	if r.Cpu >= v.Cpu {
		r.Cpu -= v.Cpu
	} else {
		r.Cpu = 0
	}
	if r.Memory >= v.Memory {
		r.Memory -= v.Memory
	} else {
		r.Memory = 0
	}
	if r.Millis >= v.Millis {
		r.Millis -= v.Millis
	} else {
		r.Millis = 0
	}
	return r
}

// Merge treats the receiver and argument as describing resource limits and
// returns the resources describing the intersection of those limits.
func (r RuntimeResources) Merge(r1 RuntimeResources) RuntimeResources {
	if smallerLimit(r1.Cpu, r.Cpu) {
		r.Cpu = r1.Cpu
	}
	if smallerLimit(r1.Memory, r.Memory) {
		r.Memory = r1.Memory
	}
	if smallerLimit(r1.Millis, r.Millis) {
		r.Millis = r1.Millis
	}
	return r
}

// Dominates returns true if the resource count v doesn't reach the resource
// limit r.
func (r RuntimeResources) Dominates(v RuntimeResources) bool {
	return !atLimit(v.Cpu, r.Cpu) && !atLimit(v.Memory, r.Memory) && !atLimit(v.Millis, r.Millis)
}

// n < m, but with 0 meaning +infinity for both n and m
func smallerLimit(n, m uint64) bool {
	return n > 0 && (m == 0 || n < m)
}

// l <= v, but with 0 meaning +infinity for l
func atLimit(v, l uint64) bool {
	return l > 0 && v >= l
}

type GCPolicy int16

const (
	DefaultGCPolicy GCPolicy = iota
	ShareGCPolicy
	IsolateGCPolicy
	UnknownGCPolicy
)
