package src

import (
	"errors"
	"fmt"
	"sync"
)

///////////////////////////////////////////////
// Initial a new virtual memory with given memory size and page size
///////////////////////////////////////////////
func NewVM(memorySize int, pageByteSize int) *VM {
	vm := new(VM)
	vm.PAGE_BYTESIZE = pageByteSize
	vm.Memory = make([]byte, max(memorySize, pageByteSize))
	vm.AccessMap = make(map[int]byte)
	vm.FreeMemory = make([]AddrPair, 1)
	vm.FreeMemory[0] = AddrPair{0, max(memorySize, pageByteSize) - 1}
	vm.mallocHistory = make(map[int]int)
	vm.accessRightDisable = false //access rights disable
	vm.faultListeners = make([]FaultListener, 0)
	vm.mutex = new(sync.Mutex)
	vm.accessLock = new(sync.Mutex)
	return vm
}

///////////////////////////////////////////////
// memory allocate
// if freeMemory has enough space, update the history map and free memory
///////////////////////////////////////////////

func (m *VM) Malloc(sizeInBytes int) (int, error) {
	for i, pair := range m.FreeMemory {
		if sizeInBytes <= pair.End-pair.Start+1 {
			m.FreeMemory[i] = AddrPair{pair.Start + sizeInBytes, pair.End}
			m.mallocHistory[pair.Start] = sizeInBytes
			return pair.Start, nil
		}
	}
	return -1, errors.New("Error: No enough space in free memory")
}

///////////////////////////////////////////////
// free memory
// free the memory space based on the start addr
///////////////////////////////////////////////

func (m *VM) Free(start int) error {
	size := m.mallocHistory[start]
	end := start + size - 1
	var newlist []AddrPair

	if size == 0 {
		return errors.New("Error: No corresponding malloc found")
	}

	j := -1
	for i, pair := range m.FreeMemory {
		if end+1 < pair.Start {
			j = i
			break
		} else if pair.End+1 < start {
			newlist = append(newlist, pair)
		} else {
			start = min(start, pair.Start)
			end = max(end, pair.End)
		}
	}
	newlist = append(newlist, AddrPair{start, end})
	if j != -1 {
		newlist = append(newlist, m.FreeMemory[j:]...)
	}
	m.FreeMemory = newlist
	return nil
}

func (m *VM) Size() int {
	return len(m.Memory)
}

func (m *VM) PageSize() int {
	return m.PAGE_BYTESIZE
}

///////////////////////////////////////////////
// read memory in page
// 1. if access rights disabled, just read
// 2. if access right is NO_ACCESS, run page fault
///////////////////////////////////////////////
func (m *VM) Read(addr int) (byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.accessRightDisable {
		return m.Memory[addr], nil
	}
	access := m.GetRights(addr)

	if access == NO_ACCESS {
		for _, l := range m.faultListeners {
			err := l(addr, 1, 0, "READ", []byte{0})
			if err == nil {
				return m.Memory[addr], nil
			}
		}
		return m.Memory[addr], AccessDeniedErr
	}
	return m.Memory[addr], nil
}

///////////////////////////////////////////////
// read memory in byte
// 1. find out the corresponding pages and their access rights
// 2. if access right is NO_ACCESS, run page fault
///////////////////////////////////////////////
func (m *VM) ReadBytes(addr, length int) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	firstPageAddr := m.GetPageAddr(addr)
	lastPageAddr := m.GetPageAddr(addr + length)
	result := make([]byte, length)

	addrList := make([]int, 0)
	for tempAddr := firstPageAddr; tempAddr <= lastPageAddr; tempAddr = tempAddr + m.PageSize() {
		addrList = append(addrList, tempAddr)
	}
	access := m.GetRightsList(addrList)
Loop:
	for i := range access {
		if access[i] == NO_ACCESS {
			for _, l := range m.faultListeners {
				err := l(addr, length, 1, "READ", nil)
				if err == nil {
					break Loop
				}
			}
			return nil, AccessDeniedErr
		}
	}
	copy(result, m.Memory[addr:addr+length])
	return result, nil
}

///////////////////////////////////////////////
// write memory based on address
// 1. if access rights disabled, just write
// 2. if access right is not READ_WRITE, run page fault
///////////////////////////////////////////////

func (m *VM) Write(addr int, val byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.accessRightDisable {
		m.Memory[addr] = val
		return nil
	}
	access := m.GetRights(m.GetPageAddr(addr))
	if access != READ_WRITE {
		for _, l := range m.faultListeners {
			err := l(addr, 1, 1, "WRITE", []byte{val})
			if err == nil {
				m.Memory[addr] = val
				return nil
			}
		}
		return AccessDeniedErr
	} else {
		m.Memory[addr] = val
		return nil
	}

}

///////////////////////////////////////////////
// write a list of byte into memory based on address
// 1. find out the address ranges
// 2. if access right is not READ_WRITE, run page fault
///////////////////////////////////////////////

func (m *VM) WriteBytes(addr int, val []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	length := len(val)
	firstPageAddr := m.GetPageAddr(addr)
	lastPageAddr := m.GetPageAddr(addr + length - 1)
	addrList := make([]int, 0)
	for tempAddr := firstPageAddr; tempAddr <= lastPageAddr; tempAddr = tempAddr + m.PageSize() {
		addrList = append(addrList, tempAddr)
	}
	access := m.GetRightsList(addrList)
Loop:
	for i := range access {
		if access[i] != READ_WRITE {
			for _, l := range m.faultListeners {
				err := l(addrList[i], length, 1, "WRITE", val)
				if err == nil {
					break Loop
				}
			}
			return AccessDeniedErr
		}
	}
	copy(m.Memory[addr:addr+length], val)
	return nil
}

///////////////////////////////////////////////
// read memory in byte
// 1. return the memory in byte without considering rights
///////////////////////////////////////////////
func (m *VM) ForceRead(addr, length int) []byte {
	read := make([]byte, length)
	copy(read, m.Memory[addr:addr+length])
	return read
}

///////////////////////////////////////////////
// write memory in byte
// 1. write a list of byte into the memory without considering rights and memory size
///////////////////////////////////////////////
func (m *VM) ForceWrite(addr int, data []byte) error {
	if addr+len(data) > len(m.Memory) {
		fmt.Println(len(m.Memory))
		fmt.Println(len(m.Memory[addr : addr+len(data)]))
		fmt.Println(len(data))
	}
	copy(m.Memory[addr:addr+len(data)], data)
	return nil
}

func (m *VM) AddFaultListener(l FaultListener) {
	m.faultListeners = append(m.faultListeners, l)
}

func (m *VM) AccessRightsDisabled(b bool) {
	m.accessRightDisable = b
}

func (m *VM) GetRights(addr int) byte {
	m.accessLock.Lock()
	res := m.AccessMap[m.GetPageAddr(addr)]
	m.accessLock.Unlock()
	return res
}

func (m *VM) GetRightsList(addrList []int) []byte {
	m.accessLock.Lock()
	defer m.accessLock.Unlock()
	result := make([]byte, len(addrList))
	for i, addr := range addrList {
		result[i] = m.AccessMap[m.GetPageAddr(addr)]
	}
	return result
}

func (m *VM) SetRights(addr int, access byte) {
	m.accessLock.Lock()
	m.AccessMap[m.GetPageAddr(addr)] = access
	m.accessLock.Unlock()
}

func (m *VM) SetRightsList(addrList []int, access byte) {
	m.accessLock.Lock()
	defer m.accessLock.Unlock()
	for _, addr := range addrList {
		m.AccessMap[m.GetPageAddr(addr)] = access
	}
}

func (m *VM) GetPageAddr(addr int) int {
	return addr - addr%m.PAGE_BYTESIZE
}

///////////////////////////////////////////////
// Utility
///////////////////////////////////////////////

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}
