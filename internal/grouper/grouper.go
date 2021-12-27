package grouper

import (
	"math/bits"

	"github.com/yistabraq/qframe/internal/column"
	"github.com/yistabraq/qframe/internal/index"
	"github.com/yistabraq/qframe/internal/math/integer"
)

/*
This package implements a basic hash table used for GroupBy and Distinct operations.

Hashing is done using Go runtime memhash, collisions are handled using linear probing.

When the table reaches a certain load factor it will be reallocated into a new, larger table.
*/

// An entry in the hash table. For group by operations a slice of all positions each group
// are stored. For distinct operations only the first position is stored to avoid some overhead.
type tableEntry struct {
	ix       index.Int
	hash     uint32
	firstPos uint32
	occupied bool
}

type table struct {
	entries     []tableEntry
	comparables []column.Comparable
	stats       GroupStats
	loadFactor  float64
	groupCount  uint32
	collectIx   bool
}

const growthFactor = 2

func (t *table) grow() {
	newLen := uint32(growthFactor * len(t.entries))
	newEntries := make([]tableEntry, newLen)
	bitMask := newLen - 1
	for _, e := range t.entries {
		for pos := e.hash & bitMask; ; pos = (pos + 1) & bitMask {
			if !newEntries[pos].occupied {
				newEntries[pos] = e
				break
			}
			t.stats.RelocationCollisions++
		}
	}

	t.stats.RelocationCount++
	t.entries = newEntries
	t.loadFactor = t.loadFactor / growthFactor
}

func (t *table) hash(i uint32) uint32 {
	hashVal := uint64(0)
	for _, c := range t.comparables {
		hashVal = c.Hash(i, hashVal)
	}

	return uint32(hashVal)
}

const maxLoadFactor = 0.5

func (t *table) insertEntry(i uint32) {
	if t.loadFactor > maxLoadFactor {
		t.grow()
	}

	hashSum := t.hash(i)
	bitMask := uint64(len(t.entries) - 1)
	startPos := uint64(hashSum) & bitMask
	var dstEntry *tableEntry
	for pos := startPos; dstEntry == nil; pos = (pos + 1) & bitMask {
		e := &t.entries[pos]
		if !e.occupied || e.hash == hashSum && equals(t.comparables, i, e.firstPos) {
			dstEntry = e
		} else {
			t.stats.InsertCollisions++
		}
	}

	// Update entry
	if !dstEntry.occupied {
		// Eden entry
		dstEntry.hash = hashSum
		dstEntry.firstPos = i
		dstEntry.occupied = true
		t.groupCount++
		t.loadFactor = float64(t.groupCount) / float64(len(t.entries))
	} else {
		// Existing entry
		if t.collectIx {
			// Small hack to reduce number of allocations under some circumstances. Delay
			// creation of index slice until there are at least two entries in the group
			// since we store the first position in a separate variable on the entry anyway.
			if dstEntry.ix == nil {
				dstEntry.ix = index.Int{dstEntry.firstPos, i}
			} else {
				dstEntry.ix = append(dstEntry.ix, i)
			}
		}
	}
}

func newTable(sizeExp int, comparables []column.Comparable, collectIx bool) *table {
	return &table{
		entries:     make([]tableEntry, integer.Pow2(sizeExp)),
		comparables: comparables,
		collectIx:   collectIx}
}

func equals(comparables []column.Comparable, i, j uint32) bool {
	for _, c := range comparables {
		if c.Compare(i, j) != column.Equal {
			return false
		}
	}
	return true
}

type GroupStats struct {
	RelocationCount      int
	RelocationCollisions int
	InsertCollisions     int
	GroupCount           int
	LoadFactor           float64
}

func calculateInitialSizeExp(ixLen int) int {
	// Size is expressed as 2^x to keep the size a multiple of two.
	// Initial size is picked fairly arbitrarily at the moment, we don't really know the distribution of
	// values within the index. Guarantee a minimum initial size of 8 (2³) for sanity.
	fitSize := uint64(ixLen) / 4
	return integer.Max(bits.Len64(fitSize), 3)
}

func groupIndex(ix index.Int, comparables []column.Comparable, collectIx bool) ([]tableEntry, GroupStats) {
	initialSizeExp := calculateInitialSizeExp(len(ix))
	table := newTable(initialSizeExp, comparables, collectIx)
	for _, i := range ix {
		table.insertEntry(i)
	}

	stats := table.stats
	stats.LoadFactor = table.loadFactor
	stats.GroupCount = int(table.groupCount)
	return table.entries, stats
}

func GroupBy(ix index.Int, comparables []column.Comparable) ([]index.Int, GroupStats) {
	entries, stats := groupIndex(ix, comparables, true)
	result := make([]index.Int, 0, stats.GroupCount)
	for _, e := range entries {
		if e.occupied {
			if e.ix == nil {
				result = append(result, index.Int{e.firstPos})
			} else {
				result = append(result, e.ix)
			}
		}
	}

	return result, stats
}

func Distinct(ix index.Int, comparables []column.Comparable) index.Int {
	entries, stats := groupIndex(ix, comparables, false)
	result := make(index.Int, 0, stats.GroupCount)
	for _, e := range entries {
		if e.occupied {
			result = append(result, e.firstPos)
		}
	}

	return result
}
