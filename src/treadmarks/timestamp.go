package src

import (
	"fmt"
	"unsafe"
)

type Timestamp []int32

func NewTimestamp(nrProcs uint8) Timestamp {
	ts := Timestamp(make([]int32, nrProcs))
	return ts
}

func (t Timestamp) increment(procId uint8) Timestamp {
	ts := Timestamp(make([]int32, len(t)))
	copy(ts, t)
	ts[procId]++
	return ts
}

func (t Timestamp) close() Timestamp {
	ts := Timestamp(make([]int32, len(t)))
	copy(ts, t)
	return t
}

//yy: all t's timestamp larger than o's
func (t Timestamp) larger(o Timestamp) bool {
	if len(t) != len(o) {
		panic(fmt.Sprint("Compared two timestamps of different length:\n", t, "\n", o))
	}
	for i := 0; i < len(t); i++ {
		if t[i] < o[i] {
			return false
		}
	}
	return true
}

func (t Timestamp) equals(o Timestamp) bool {
	if len(t) != len(o) {
		panic(fmt.Sprint("Compared two timestamps of different length:\n", t, "\n", o))
	}
	for i := 0; i < len(t); i++ {
		if t[i] != o[i] {
			fmt.Sprint("i = ", i, " t[i] = ", t[i], " o[i] = ", o[i])
			return false
		}
	}
	return true
}

func (t Timestamp) merge(o Timestamp) Timestamp {
	newTs := Timestamp(make([]int32, len(t)))
	if len(t) != len(o) {
		panic(fmt.Sprint("Length were not matching", len(t), " != ", len(o)))
	}
	for i := range t {
		if t[i] > o[i] {
			newTs[i] = t[i]
		} else {
			newTs[i] = o[i]
		}
	}
	return newTs
}

func (t Timestamp) min(o Timestamp) Timestamp {
	newTs := Timestamp(make([]int32, len(t)))
	for i := range t {
		if t[i] < o[i] {
			newTs[i] = t[i]
		} else {
			newTs[i] = o[i]
		}
	}
	return newTs
}

func (t Timestamp) loadFromData(data []byte) {
	for i := range t {
		t[i] = BytesToInt32(data[i*4 : i*4+4])
	}
}

func (t Timestamp) saveToData() []byte {
	data := make([]byte, 4*len(t))
	for i := range t {
		copy(data[i*4:i*4+3], Int32ToBytes(t[i]))
	}
	return data
}

func BytesToInt32(data []byte) int32 {
	return *(*int32)(unsafe.Pointer(&data[0]))
}

func Int32ToBytes(i int32) []byte {
	ptr := uintptr(unsafe.Pointer(&i))
	slice := make([]byte, 4)
	for i := 0; i < 4; i++ {
		slice[i] = *(*byte)(unsafe.Pointer(ptr))
		ptr++
	}
	return slice
}
