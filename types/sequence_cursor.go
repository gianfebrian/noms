package types

import (
	"sort"

	"github.com/attic-labs/noms/d"
)

type sequenceItem interface{}

type sequence interface {
	Value
	getItem(idx int) sequenceItem
	seqLen() int
}

// sequenceCursor explores a tree of sequence items.
type sequenceCursor struct {
	parent *sequenceCursor
	seq    sequence
	idx    int
}

func newSequenceCursor(parent *sequenceCursor, seq sequence, idx int) *sequenceCursor {
	return &sequenceCursor{parent, seq, idx}
}

func (cur *sequenceCursor) length() int {
	return cur.seq.seqLen()
}

func (cur *sequenceCursor) getItem(idx int) sequenceItem {
	return cur.seq.getItem(idx)
}

func (cur *sequenceCursor) sync() {
	d.Chk.NotNil(cur.parent)
	cur.seq = cur.parent.getChildSequence()
}

func (cur *sequenceCursor) getChildSequence() sequence {
	if ms, ok := cur.seq.(metaSequence); ok {
		return ms.getChildSequence(cur.idx)
	}
	return nil
}

// Returns the value the cursor refers to. Fails an assertion if the cursor doesn't point to a value.
func (cur *sequenceCursor) current() sequenceItem {
	item, ok := cur.maybeCurrent()
	d.Chk.True(ok)
	return item
}

func (cur *sequenceCursor) valid() bool {
	return cur.idx >= 0 && cur.idx < cur.length()
}

// Returns the value the cursor refers to, if any. If the cursor doesn't point to a value, returns (nil, false).
func (cur *sequenceCursor) maybeCurrent() (sequenceItem, bool) {
	d.Chk.True(cur.idx >= -1 && cur.idx <= cur.length())
	if !cur.valid() {
		return nil, false
	}
	return cur.getItem(cur.idx), true
}

func (cur *sequenceCursor) indexInChunk() int {
	return cur.idx
}

func (cur *sequenceCursor) advance() bool {
	return cur.advanceMaybeAllowPastEnd(true)
}

func (cur *sequenceCursor) advanceMaybeAllowPastEnd(allowPastEnd bool) bool {
	if cur.idx < cur.length()-1 {
		cur.idx++
		return true
	}
	if cur.idx == cur.length() {
		return false
	}
	if cur.parent != nil && cur.parent.advanceMaybeAllowPastEnd(false) {
		cur.sync()
		cur.idx = 0
		return true
	}
	if allowPastEnd {
		cur.idx++
	}
	return false
}

func (cur *sequenceCursor) retreat() bool {
	return cur.retreatMaybeAllowBeforeStart(true)
}

func (cur *sequenceCursor) retreatMaybeAllowBeforeStart(allowBeforeStart bool) bool {
	if cur.idx > 0 {
		cur.idx--
		return true
	}
	if cur.idx == -1 {
		return false
	}
	d.Chk.Equal(0, cur.idx)
	if cur.parent != nil && cur.parent.retreatMaybeAllowBeforeStart(false) {
		cur.sync()
		cur.idx = cur.length() - 1
		return true
	}
	if allowBeforeStart {
		cur.idx--
	}
	return false
}

func (cur *sequenceCursor) clone() *sequenceCursor {
	var parent *sequenceCursor
	if cur.parent != nil {
		parent = cur.parent.clone()
	}
	return &sequenceCursor{parent, cur.seq, cur.idx}
}

type cursorIterCallback func(item interface{}) bool

func (cur *sequenceCursor) iter(cb cursorIterCallback) {
	for cur.valid() && !cb(cur.getItem(cur.idx)) {
		cur.advance()
	}
}

type sequenceCursorSeekBinaryCompareFn func(item sequenceItem) bool

// seekBinary seeks the cursor to the first position in the sequence where |compare| returns true. This uses a binary search, so the cursor items must be sorted relative to |compare|. seekBinary will not seek past the end of the cursor.
func (cur *sequenceCursor) seekBinary(compare sequenceCursorSeekBinaryCompareFn) {
	d.Chk.NotNil(compare)

	if cur.parent != nil {
		cur.parent.seekBinary(compare)
		cur.sync()
	}

	cur.idx = sort.Search(cur.length(), func(i int) bool {
		return compare(cur.getItem(i))
	})

	if cur.idx == cur.length() {
		cur.idx = cur.length() - 1
	}
}

// Returns a slice of the previous |n| items in |cur|, excluding the current item in |cur|. Does not modify |cur|.
func (cur *sequenceCursor) maxNPrevItems(n int) []sequenceItem {
	prev := []sequenceItem{}

	retreater := cur.clone()
	for i := 0; i < n && retreater.retreat(); i++ {
		prev = append(prev, retreater.current())
	}

	for i := 0; i < len(prev)/2; i++ {
		t := prev[i]
		prev[i] = prev[len(prev)-i-1]
		prev[len(prev)-i-1] = t
	}

	return prev
}
