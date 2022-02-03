package runtime

import (
	"runtime"
	"sort"
	"sync"
	"unsafe"
)

type WeakRef struct {
	tp      ValueType
	ptr     uintptr
	gcOrder int
}

func (r *WeakRef) Value() Value {
	if r.ptr == 0 {
		return NilValue
	}
	switch r.tp {
	case TableType:
		return TableValue((*Table)(unsafe.Pointer(r.ptr)))
	case UserDataType:
		return UserDataValue((*UserData)(unsafe.Pointer(r.ptr)))
	default:
		return NilValue
	}
}

type WeakRefPool struct {
	weakrefs  map[uintptr]*WeakRef
	finalized []interface{}
	mx        sync.Mutex
	gcOrder   int
}

func newWeakRefPool() *WeakRefPool {
	return &WeakRefPool{weakrefs: make(map[uintptr]*WeakRef)}
}

func (p *WeakRefPool) Get(v Value) *WeakRef {
	var tp ValueType
	switch v.iface.(type) {
	case *Table:
		tp = TableType
	case *UserData:
		tp = UserDataType
	default:
		return nil
	}
	ptr := ifacePtr(v.iface)
	r := p.weakrefs[ptr]
	if r == nil {
		runtime.SetFinalizer(v.iface, p.addPending)
		r = &WeakRef{tp: tp, ptr: ptr}
		p.weakrefs[ptr] = r
	}
	return r
}

func (p *WeakRefPool) SetGC(v Value) {
	switch v.iface.(type) {
	case *Table, *UserData:
		// s, _ := v.ToString()
		// log.Printf("set gc for %s", s)
		p.gcOrder++
		p.Get(v).gcOrder = p.gcOrder
	}
}

func (p *WeakRefPool) ExtractPendingGC() []Value {
	p.mx.Lock()
	finalized := p.finalized
	p.finalized = nil
	p.mx.Unlock()
	if len(finalized) == 0 {
		return nil
	}
	var pendingGC []Value
	var orders []int
	// log.Printf("extract from %d", len(finalized))
	for _, iface := range finalized {
		r := p.weakrefs[ifacePtr(iface)]
		if r != nil {
			r.ptr = 0
			if r.gcOrder > 0 {
				pendingGC = append(pendingGC, AsValue(iface))
				orders = append(orders, r.gcOrder)
			}
		}
	}
	// log.Printf("got %d", len(pendingGC))
	// Lua wants to run finalizers in reverse order
	sort.Slice(pendingGC, func(i, j int) bool { return orders[i] > orders[j] })
	return pendingGC
}

func (p *WeakRefPool) GetAllGC() []Value {
	var vals []Value
	var orders []int
	for _, r := range p.weakrefs {
		if r.gcOrder > 0 {
			v := r.Value()
			if !v.IsNil() {
				vals = append(vals, r.Value())
				orders = append(orders, r.gcOrder)
			}
		}
	}
	// Sort in reverse order
	sort.Slice(vals, func(i, j int) bool { return orders[i] > orders[j] })
	return vals
}

func (p *WeakRefPool) addPending(iface interface{}) {
	// log.Printf("add pending %+v", iface)
	p.mx.Lock()
	defer p.mx.Unlock()
	p.finalized = append(p.finalized, iface)
}
