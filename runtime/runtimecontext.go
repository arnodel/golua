package runtime

type RuntimeContextStatus uint8

const (
	RCS_Live RuntimeContextStatus = iota
	RCS_Done
	RCS_Killed
)

type RuntimeContext interface {
	CpuLimit() uint64
	CpuUsed() uint64

	MemLimit() uint64
	MemUsed() uint64

	Status() RuntimeContextStatus
	Parent() RuntimeContext
}
