package runtime

type RuntimeContext interface {
	CpuLimit() uint64
	CpuUsed() uint64

	MemLimit() uint64
	MemUsed() uint64

	Status() RuntimeContextStatus
	Parent() RuntimeContext

	SafetyFlags() ComplianceFlags
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

type RuntimeContextDef struct {
	CpuLimit    uint64
	MemLimit    uint64
	SafetyFlags ComplianceFlags
}
