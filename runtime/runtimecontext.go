package runtime

type RuntimeContext interface {
	HardResourceLimits() RuntimeResources
	SoftResourceLimits() RuntimeResources
	UsedResources() RuntimeResources

	Status() RuntimeContextStatus
	Parent() RuntimeContext

	SafetyFlags() ComplianceFlags

	ShouldCancel() bool
}

type RuntimeContextStatus uint8

const (
	RCS_Live RuntimeContextStatus = iota
	RCS_Done
	RCS_Killed
)

type ComplianceFlags uint16

const (
	ComplyMemSafe ComplianceFlags = 1 << iota
	ComplyCpuSafe
	ComplyIoSafe
	complyflagsLimit
)

var flagNames = map[ComplianceFlags]string{
	ComplyMemSafe: "memsafe",
	ComplyCpuSafe: "cpusafe",
	ComplyIoSafe:  "iosafe",
}

var nameFlags = map[string]ComplianceFlags{
	"memsafe": ComplyMemSafe,
	"cpusafe": ComplyCpuSafe,
	"iosafe":  ComplyIoSafe,
}

func (f ComplianceFlags) AddFlagWithName(name string) (ComplianceFlags, bool) {
	fn, ok := nameFlags[name]
	return fn | f, ok
}

func (f ComplianceFlags) Names() (names []string) {
	var i ComplianceFlags
	for i = 1; i < complyflagsLimit; i <<= 1 {
		if i&f != 0 {
			names = append(names, flagNames[i])
		}
	}
	return names
}

type RuntimeResources struct {
	Cpu   uint64
	Mem   uint64
	Timer uint64
}

func (r RuntimeResources) Remove(r1 RuntimeResources) RuntimeResources {
	if r.Cpu >= r1.Cpu {
		r.Cpu -= r1.Cpu
	}
	if r.Mem >= r1.Mem {
		r.Mem -= r1.Mem
	}
	if r.Timer >= r1.Timer {
		r.Timer -= r1.Timer
	}
	return r
}

func (r RuntimeResources) Merge(r1 RuntimeResources) RuntimeResources {
	if r.Cpu == 0 || r.Cpu > r1.Cpu {
		r.Cpu = r1.Cpu
	}
	if r.Mem == 0 || r.Mem > r1.Mem {
		r.Mem = r1.Mem
	}
	if r.Timer == 0 || r.Timer > r1.Timer {
		r.Timer = r1.Timer
	}
	return r
}

func (r RuntimeResources) Dominates(r1 RuntimeResources) bool {
	if r.Cpu > 0 && r1.Cpu >= r.Cpu {
		return false
	}
	if r.Mem > 0 && r1.Mem >= r.Mem {
		return false
	}
	if r.Timer > 0 && r1.Timer >= r.Timer {
		return false
	}
	return true
}

type RuntimeContextDef struct {
	HardLimits  RuntimeResources
	SoftLimits  RuntimeResources
	SafetyFlags ComplianceFlags
}
