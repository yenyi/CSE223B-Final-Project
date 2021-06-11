package src

import (
	"sync"
	"github.com/DistributedClocks/GoVector/govec"
)

type TreadMarksAPI interface {
	Initialize(port int) error
	Join(ip string, port int) error
	Shutdown() error
	Read(addr int) (byte, error)
	Write(addr int, val byte) error
	ReadBytes(addr int, length int) ([]byte, error)
	WriteBytes(addr int, val []byte) error
	Malloc(size int) (int, error)
	Free(addr, size int) error
	Barrier(id uint8)
	AcquireLock(id uint8)
	ReleaseLock(id uint8)
	GetId() int
}


type TreadMarks struct {
	shutdown                       chan bool
	memory                         VirtualMemory
	Id				               uint8
	numProcs					   uint8
	memSize, pageByteSize, nrPages int
	pageArray                      []*pageArrayEntry
	twins                          [][]byte
	twinsLock                      *sync.RWMutex
	dirtyPages                     map[int16]bool
	dirtyPagesLock                 *sync.RWMutex
	procArray                      [][]IntervalRecord
	locks                          []*lock
	channel                        chan bool
	barrier                        chan uint8
	barrierreq                     []BarrierRequest
	in                             <-chan []byte
	out                            chan<- []byte
	conn                           Connection
	group                          *sync.WaitGroup
	timestamp                      Timestamp
	diffLock                       *sync.Mutex
	messageLog                     []int
	vecLog     					   *govec.GoLog      // GoVector object
}

const (
	LOCKREQ = 0
	LOCKRSP = 1
	BARRREQ = 2
	BARRRSP = 3
	COPYREQ = 4 
	COPYRSP = 5
	DIFFREQ = 6 
	DIFFRSP = 7 
)

var msgName = map[uint8]string{
	LOCKREQ: "LOCKREQ", LOCKRSP: "LOCKRSP", BARRREQ: "BARRREQ", BARRRSP: "BARRRSP",
	COPYREQ: "COPYREQ", COPYRSP: "COPYRSP", DIFFREQ: "DIFFREQ", DIFFRSP: "DIFFRSP",
}
