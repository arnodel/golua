package runtime

import (
	"testing"
)

func v(i interface{}) Value {
	return AsValue(i)
}

func TestHashTable(t *testing.T) {
	var ht *hashTable
	if !ht.full() {
		t.Error("Expected nil hashTable to be full")
	}
	ht = ht.grow()
	if ht.full() {
		t.Error("Expected growed hashTable not to be full")
	}
	ht.set(v("hello"), v(123))
	if ht.find(v("hello")) != v(123) {
		t.Error("expected hello => 123")
	}
	if !ht.full() {
		t.Error("should be full")
	}
}

func TestMixedTable1To20(t *testing.T) {
	var mt = new(mixedTable)
	const n = 20
	for i := 1; i <= n; i++ {
		mt.insert(v(i), v(i+2))
	}
	for i := 1; i <= n; i++ {
		mti := mt.get(v(i))
		if mti != v(i+2) {
			t.Errorf("Expected mt[%d] to be %d, got %v", i, i+2, mti)
		}
	}
	asz := mt.array.size()
	if asz != 32 {
		t.Errorf("expected array size to be 32, got %d", asz)
	}
	alen := mt.array.getLen()
	if alen != 20 {
		t.Errorf("Expected array length to be 20, got %d", alen)
	}
	mtlen := mt.len()
	if mtlen != 20 {
		t.Errorf("Expected mt to have length 20, got %d", mtlen)
	}
}

func TestMixedTableRemove(t *testing.T) {
	mt := new(mixedTable)
	for i := 1; i <= 5; i++ {
		mt.insert(v(i), v(i))
	}
	mt.remove(v(5))
	mt.remove(v(4))
	alen := mt.array.getLen()
	if alen != 3 {
		t.Errorf("Expected array length to be 3, got %d", alen)
	}
	mtlen := mt.len()
	if mtlen != 3 {
		t.Errorf("Expected mt length to be 3, got %d", mtlen)
	}
}

func Test_calculateArraySize(t *testing.T) {
	type args struct {
		idxCountByLen *[uintptrLen]uintptr
	}
	var counts = func(cs ...uintptr) *[uintptrLen]uintptr {
		counts := new([uintptrLen]uintptr)
		for i, c := range cs {
			counts[i] = c
		}
		return counts
	}

	tests := []struct {
		name string
		args args
		want uintptr
	}{
		{
			name: "empty",
			args: args{
				idxCountByLen: counts(),
			},
			want: 0,
		},
		{
			name: "0",
			args: args{
				idxCountByLen: counts(1),
			},
			want: 1,
		},
		{
			name: "0 1",
			args: args{
				idxCountByLen: counts(1, 1),
			},
			want: 2,
		},
		{
			name: "1",
			args: args{
				idxCountByLen: counts(0, 1),
			},
			want: 2,
		},
		{
			name: "0 1 2",
			args: args{
				idxCountByLen: counts(1, 1, 1),
			},
			want: 4,
		},
		{
			name: "0 1 2 3",
			args: args{
				idxCountByLen: counts(1, 1, 2),
			},
			want: 4,
		},
		{
			name: "1 2 3 4",
			args: args{
				idxCountByLen: counts(0, 1, 2, 1),
			},
			want: 8,
		},

		{
			name: "0..15",
			args: args{
				idxCountByLen: counts(1, 1, 2, 4, 8),
			},
			want: 16,
		},
		{
			name: "0..16",
			args: args{
				idxCountByLen: counts(1, 1, 2, 4, 8, 1),
			},
			want: 32,
		},
		{
			name: "3..18",
			args: args{
				idxCountByLen: counts(0, 0, 1, 4, 8, 3),
			},
			want: 32,
		},
		{
			name: "10..49",
			args: args{
				idxCountByLen: counts(0, 0, 0, 0, 6, 16, 18),
			},
			want: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateArraySize(tt.args.idxCountByLen); got != tt.want {
				t.Errorf("calculateArraySize() = %v, want %v", got, tt.want)
			}
		})
	}
}
