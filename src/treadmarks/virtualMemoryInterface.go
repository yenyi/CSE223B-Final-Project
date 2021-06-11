package src

import (
	"errors"
	"sync"
)

var AccessDeniedErr = errors.New("access denied")

const (
	NO_ACCESS byte = 0
 	READ_ONLY byte = 1
 	READ_WRITE byte = 2
)

type FaultListener func(addr int, length int, faultType byte, accessType string, value []byte) error //faultType = 0 => read fault, = 1 => write fault

type VirtualMemory interface {
	Malloc(sizeInBytes int) (int, error)
	Free(start int) error
	Size() int
	PageSize() int
	Read(addr int) (byte, error)
	ReadBytes(addr, length int) ([]byte, error)
	Write(addr int, val byte) error
	WriteBytes(addr int, val []byte) error
	ForceRead(addr, length int) []byte
	ForceWrite(addr int, data []byte) error
	GetRights(addr int) byte
	GetRightsList(addr []int) []byte
	SetRights(addr int, access byte)
	SetRightsList(addr []int, access byte)
	GetPageAddr(addr int) int
	AccessRightsDisabled(b bool)
	AddFaultListener(l FaultListener)
}

type VM struct {
	Memory             []byte
	AccessMap          map[int]byte
	PAGE_BYTESIZE      int
	FreeMemory         []AddrPair
	mallocHistory      map[int]int //maps address to length
	accessRightDisable bool
	faultListeners     []FaultListener
	accessLock         *sync.Mutex

	mutex *sync.Mutex
}

type AddrPair struct {
	Start, End int
}
