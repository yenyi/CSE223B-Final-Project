package src

import (
	"sync"
)

type data []byte

type IntervalRecord struct {
	Owner     uint8
	Timestamp Timestamp
	Pages     []int16
}

type WritenoticeRecord struct {
	Owner     uint8
	Timestamp Timestamp
	Diff      map[int]byte
}

type pageArrayEntry struct {
	curPos           []int
	hasMissingDiffs bool
	hasCopy         bool
	copySet         []uint8
	writenotices    [][]WritenoticeRecord
}

type lock struct {
	sync.Locker
	locked        bool
	haveToken     bool
	last          uint8
	nextId        uint8
	nextTimestamp Timestamp
}

type LockAcquireRequest struct {
	From      uint8
	LockId    uint8
	Timestamp Timestamp
}

type LockAcquireResponse struct {
	LockId    uint8
	Timestamp Timestamp
	Intervals []IntervalRecord
}

type BarrierRequest struct {
	From      uint8
	BarrierId uint8
	Timestamp Timestamp
	Intervals []IntervalRecord
}

type BarrierResponse struct {
	Intervals []IntervalRecord
	Timestamp Timestamp
}

type DiffRequest struct {
	From   uint8
	to     uint8
	PageNr int16
	First  Timestamp
	Last   Timestamp
}

type DiffResponse struct {
	PageNr       int16
	Writenotices []WritenoticeRecord
}

type CopyRequest struct {
	From   uint8
	PageNr int16
}

type CopyResponse struct {
	PageNr int16
	Data   []byte
}
