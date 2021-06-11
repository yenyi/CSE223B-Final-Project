package src

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"
	"github.com/DistributedClocks/GoVector/govec"
)



var _ TreadMarksAPI = new(TreadMarks)

///////////////////////////////
// Initialize
///////////////////////////////

func newPageArrayEntry(numProcs uint8) *pageArrayEntry {
	wnl := make([][]WritenoticeRecord, numProcs)
	for j := range wnl {
		wnl[j] = make([]WritenoticeRecord, 0)
	}
	entry := &pageArrayEntry{
		curPos:        make([]int, numProcs),
		hasCopy:      false,
		copySet:      []uint8{0},
		writenotices: wnl,
	}
	return entry
}

func newPageArray(numPages int, numProcs uint8) []*pageArrayEntry {
	array := make([]*pageArrayEntry, numPages)
	for i := range array {
		array[i] = newPageArrayEntry(numProcs)
	}
	return array
}



func newProcArray(numProcs uint8) [][]IntervalRecord {
	procArray := make([][]IntervalRecord, numProcs)
	for i := range procArray {
		procArray[i] = make([]IntervalRecord, 0)
	}
	return procArray
}

func (tm *TreadMarks) faultHandler(addr int, length int, faultType byte, accessType string, value []byte) error {
	addrList := make([]int, 0)
	for i := tm.memory.GetPageAddr(addr); i < addr+length; i = i + tm.memory.PageSize() {
		addrList = append(addrList, i)
	}
	access := tm.memory.GetRightsList(addrList)

	for i := range access {
		pageNr := int16(math.Floor(float64(tm.memory.GetPageAddr(addr)) / float64(tm.memory.PageSize())))
		page := tm.pageArray[pageNr]
		if access[i] == NO_ACCESS {
			// first time access that page, i.e., missing page
			if !page.hasCopy {
				tm.sendCopyRequest(pageNr)
				<-tm.channel
			}
			if tm.hasMissingDiffs(pageNr) {
				tm.sendDiffRequests(pageNr)
				tm.putDiffs(pageNr)
			}
		}
		if accessType == "WRITE" {
			tm.twinsLock.Lock()
			tm.dirtyPagesLock.Lock()

			tm.twins[pageNr] = make([]byte, tm.pageByteSize)

			copy(tm.twins[pageNr], tm.memory.ForceRead(tm.memory.GetPageAddr(int(pageNr)), tm.memory.PageSize()))
			tm.dirtyPages[pageNr] = true
			tm.dirtyPagesLock.Unlock()
			tm.twinsLock.Unlock()
		}
	}

	if accessType == "WRITE" {
		tm.memory.SetRightsList(addrList, READ_WRITE)
	} else {
		tm.memory.SetRightsList(addrList, READ_ONLY)
	}
	return nil
}

func (tm *TreadMarks) initializeLocks() {
	for i := uint8(0); i < uint8(len(tm.locks)); i++ {
		tm.locks[i] = &lock{new(sync.Mutex), false, tm.Id == tm.managerID(i), tm.managerID(i), 0, nil}
	}
}

func (tm *TreadMarks) initializeBarriers() {
	tm.barrier = make(chan uint8, 1)
	tm.barrier <- 0
}

func NewTreadMarks(memSize, pageByteSize int, numProcs, nrLocks, nrBarriers uint8) (*TreadMarks, error) {
	var err error
	tm := new(TreadMarks)
	tm.memSize, tm.pageByteSize, tm.numProcs = memSize, pageByteSize, numProcs
	tm.memory = NewVM(memSize, pageByteSize)
	tm.nrPages = int(math.Ceil(float64(memSize) / float64(pageByteSize)))
	tm.pageArray = newPageArray(tm.nrPages, numProcs)
	tm.procArray = newProcArray(numProcs)
	tm.locks = make([]*lock, nrLocks)
	tm.channel = make(chan bool, 1)

	tm.barrierreq = make([]BarrierRequest, tm.numProcs)
	tm.timestamp = NewTimestamp(tm.numProcs)
	tm.twins = make([][]byte, tm.nrPages)
	tm.twinsLock = new(sync.RWMutex)
	tm.dirtyPages = make(map[int16]bool)
	tm.dirtyPagesLock = new(sync.RWMutex)
	tm.diffLock = new(sync.Mutex)

	return tm, err
}


///////////////////////////////
// Interface Functions
///////////////////////////////
func (tm *TreadMarks) Initialize(port int) error {
	tm.conn, tm.in, tm.out, _ = NewConnection(port, 10)
	tm.memory.AddFaultListener(tm.faultHandler)
	tm.group = new(sync.WaitGroup)
	tm.initializeLocks()
	tm.initializeBarriers()
	process := "Proc:" + strconv.Itoa(int(port))
	tm.vecLog = govec.InitGoVector(process, process, govec.GetDefaultConfig())
	tm.shutdown = make(chan bool)
	go tm.messageHandler()
	tm.messageLog = make([]int, 9)
	return nil
}

func (tm *TreadMarks) Join(ip string, port int) error {
	id, err := tm.conn.Connect(ip, port)
	if err != nil {
		return err
	}
	tm.Id = uint8(id)
	tm.initializeLocks()
	tm.timestamp = NewTimestamp(tm.numProcs)
	return nil
}

func (tm *TreadMarks) Shutdown() error {
	fmt.Println("Lock acquire request messages: ", tm.messageLog[0])
	fmt.Println("Lock acquire response messages: ", tm.messageLog[1])
	fmt.Println("Barrier request messages: ", tm.messageLog[2])
	fmt.Println("Barrier response messages: ", tm.messageLog[3])
	fmt.Println("Copy request messages: ", tm.messageLog[4])
	fmt.Println("Copy response messages: ", tm.messageLog[5])
	fmt.Println("Diffs request messages: ", tm.messageLog[6])
	fmt.Println("Diffs response messages: ", tm.messageLog[7])
	tm.shutdown <- true
	tm.group.Wait()
	tm.conn.Close()

	return nil
}

func (tm *TreadMarks) Malloc(size int) (int, error) {
	return tm.memory.Malloc(size)
}

func (tm *TreadMarks) Free(addr, size int) error {
	return tm.memory.Free(addr)
}

func (tm *TreadMarks) Barrier(id uint8) {
	tm.sendBarrierRequest(id)
}

func (tm *TreadMarks) Read(addr int) (byte, error) {
	return tm.memory.Read(addr)
}

func (tm *TreadMarks) ReadBytes(addr int, length int) ([]byte, error) {
	return tm.memory.ReadBytes(addr, length)
}

func (tm *TreadMarks) Write(addr int, val byte) error {
	return tm.memory.Write(addr, val)
}

func (tm *TreadMarks) WriteBytes(addr int, val []byte) error {
	return tm.memory.WriteBytes(addr, val)
}

func (tm *TreadMarks) AcquireLock(id uint8) {
	lock := tm.locks[id]
	lock.Lock()
	if lock.haveToken {
		if lock.locked {
			panic("Error: Already hold the lock")
		}
		lock.locked = true
		lock.Unlock()
	} else {
		tm.sendLockAcquireRequest(lock.last, id)
		lock.last = tm.Id
		lock.Unlock()
		<-tm.channel
	}
}

func (tm *TreadMarks) ReleaseLock(id uint8) {
	lock := tm.locks[id]
	lock.Lock()
	defer lock.Unlock()
	lock.locked = false
	if lock.nextTimestamp != nil {
		tm.newInterval()
		tm.sendLockAcquireResponse(id, lock.nextId, lock.nextTimestamp)
		lock.haveToken = false
		lock.nextTimestamp = nil
		lock.nextId = tm.managerID(id)
	}
}

///////////////////////////////
// Write Notices
///////////////////////////////

func (tm *TreadMarks) newWritenoticeRecord(pageNr int16) {
	ts := NewTimestamp(tm.numProcs).merge(tm.timestamp)
	wn := WritenoticeRecord{
		Owner:     tm.Id,
		Timestamp: ts,
	}
	tm.pageArray[pageNr].writenotices[tm.Id] = append(tm.pageArray[pageNr].writenotices[tm.Id], wn)
	delete(tm.dirtyPages, pageNr)
}

func (tm *TreadMarks) addWritenoticeRecord(pageNr int16, procId uint8, timestamp Timestamp) {
	pageSize := tm.memory.PageSize()
	addr := int(pageNr) * pageSize
	access := tm.memory.GetRights(addr)

	if access == READ_WRITE {
		tm.dirtyPagesLock.Lock()
		if tm.dirtyPages[pageNr] {
			tm.newWritenoticeRecord(pageNr)
		}
		tm.dirtyPagesLock.Unlock()
		tm.twinsLock.Lock()
		tm.generateDiff(pageNr, tm.twins[pageNr])
		tm.twins[pageNr] = nil
		tm.twinsLock.Unlock()
	}
	tm.memory.SetRights(addr, NO_ACCESS)
	wn := WritenoticeRecord{
		Owner:     procId,
		Timestamp: timestamp,
	}
	page := tm.pageArray[pageNr]
	wnl := page.writenotices[procId]
	wnl = append(wnl, wn)
	page.hasMissingDiffs = true
	tm.pageArray[pageNr].writenotices[procId] = wnl
}

///////////////////////////////
// Intervals
///////////////////////////////

//used when handle lock and barrier request
func (tm *TreadMarks) newInterval() {
	tm.dirtyPagesLock.Lock()
	if len(tm.dirtyPages) > 0 {
		pages := make([]int16, 0, len(tm.dirtyPages))

		tm.timestamp = tm.timestamp.increment(tm.Id)
		for page := range tm.dirtyPages {
			pages = append(pages, page)
			tm.newWritenoticeRecord(page)
		}
		ts := NewTimestamp(tm.numProcs).merge(tm.timestamp)
		interval := IntervalRecord{
			Owner:     tm.Id,
			Timestamp: ts,
			Pages:     pages,
		}
		tm.procArray[tm.Id] = append(tm.procArray[tm.Id], interval)
	}

	tm.dirtyPagesLock.Unlock()
}

func (tm *TreadMarks) addInterval(interval IntervalRecord) {
	ts := NewTimestamp(tm.numProcs).merge(tm.timestamp)
	if !ts.larger(interval.Timestamp) || !interval.Timestamp.larger(ts) {
		for _, p := range interval.Pages {
			tm.addWritenoticeRecord(p, interval.Owner, interval.Timestamp)
		}
		tm.procArray[interval.Owner] = append(tm.procArray[interval.Owner], interval)
		tm.timestamp = tm.timestamp.merge(interval.Timestamp)
	}
}

///////////////////////////////
// Diffs
///////////////////////////////
//compare current virtual memory and twin
func (tm *TreadMarks) generateDiff(pageNr int16, twin []byte) {
	pageSize := tm.memory.PageSize()
	addr := int(pageNr) * pageSize
	tm.memory.SetRights(addr, READ_ONLY)
	data := tm.memory.ForceRead(addr, pageSize)
	diff := make(map[int]byte)
	for i := range data {
		if data[i] != twin[i] {
			diff[i] = data[i]
		}
	}
	tm.pageArray[pageNr].hasMissingDiffs = false

	if len(diff) == 0 {
		tm.pageArray[pageNr].writenotices[tm.Id] = tm.pageArray[pageNr].writenotices[tm.Id][:len(tm.pageArray[pageNr].writenotices[tm.Id])-1]
	} else {
		tm.pageArray[pageNr].writenotices[tm.Id][len(tm.pageArray[pageNr].writenotices[tm.Id])-1].Diff = diff
	}
}

func (tm *TreadMarks) createDiffRequests(pageNr int16) []DiffRequest {
	diffRequests := make([]DiffRequest, 0, tm.numProcs) //yy: b := make([]int, 0, 5) // len(b)=0, cap(b)=5
	var proc uint8
	for proc = 0; proc < tm.numProcs; proc++ {
		req := tm.createDiffRequest(pageNr, proc)
		if req.Last != nil {
			insert := true
			for i, oReq := range diffRequests {
				if oReq.Last.larger(req.Last) {
					diffRequests[i].First = oReq.First.min(req.First)
					insert = false
				} else if req.Last.larger(oReq.Last) {
					diffRequests[i].to = req.to
					diffRequests[i].First = oReq.First.min(req.First)
					diffRequests[i].Last = oReq.Last.merge(req.Last)
					insert = false
				}
			}
			if insert {
				diffRequests = append(diffRequests, req)
			}
		}

	}
	return diffRequests
}

func (tm *TreadMarks) createDiffRequest(pageNr int16, procId uint8) DiffRequest {
	req := DiffRequest{
		to:     procId,
		From:   tm.Id,
		PageNr: pageNr,
	}

	wnl := tm.pageArray[pageNr].writenotices[procId]

	l := len(wnl) - 1
	if l >= 0 && wnl[l].Diff == nil {
		req.Last = wnl[l].Timestamp
		for ; l >= 0; l-- {
			if wnl[l].Diff != nil {
				break
			}
			req.First = wnl[l].Timestamp
		}
	}
	return req
}



///////////////////////////////
// Get missing...
///////////////////////////////

func (tm *TreadMarks) getMissingIntervals(ts Timestamp) []IntervalRecord {
	var proc uint8
	intervals := make([]IntervalRecord, 0, int(tm.numProcs)*5)
	for proc = 0; proc < tm.numProcs; proc++ {
		intervals = append(intervals, tm.getMissingIntervalsForProc(proc, ts)...)
	}
	return intervals
}

func (tm *TreadMarks) getMissingIntervalsForProc(procId uint8, ts Timestamp) []IntervalRecord {
	intervals := tm.procArray[procId]
	result := make([]IntervalRecord, 0, len(intervals))
	for i := len(intervals) - 1; i >= 0; i-- {
		if !ts.larger(intervals[i].Timestamp) {
			result = append(result, intervals[i])
		} else {
			break
		}
	}
	return result
}

func (tm *TreadMarks) hasMissingDiffs(pageNr int16) bool {
	return tm.pageArray[pageNr].hasMissingDiffs
}




///////////////////////////////
// Send messages
///////////////////////////////
func (tm *TreadMarks) sendMessage(to, msgType uint8, msg interface{}) {
	tm.log(msgType)

	event := "Send " + msgName[msgType]
	opt:= govec.GetDefaultLogOptions()
	gbuf := tm.vecLog.PrepareSend(event, msg, opt)
	data := make([]byte, len(gbuf)+2)
	data[0] = byte(to)
	data[1] = byte(msgType)
	copy(data[2:], gbuf)

	tm.out <- data
}

func (tm *TreadMarks) sendLockAcquireRequest(to uint8, lockId uint8) {
	req := LockAcquireRequest{
		From:      tm.Id,
		LockId:    lockId,
		Timestamp: tm.timestamp,
	}
	tm.sendMessage(to, LOCKREQ, req)
}

func (tm *TreadMarks) sendLockAcquireResponse(lockId uint8, to uint8, timestamp Timestamp) {

	intervals := tm.getMissingIntervals(timestamp)
	resp := LockAcquireResponse{
		LockId:    lockId,
		Intervals: intervals,
		Timestamp: tm.timestamp,
	}

	tm.sendMessage(to, LOCKRSP, resp)
}

func (tm *TreadMarks) forwardLockAcquireRequest(to uint8, req LockAcquireRequest) {
	tm.sendMessage(to, LOCKREQ, req)
}

func (tm *TreadMarks) sendBarrierRequest(barrierId uint8) {
	managerId := tm.managerID(barrierId)
	req := BarrierRequest{
		From:      tm.Id,
		BarrierId: barrierId,
		Timestamp: tm.timestamp,
	}
	if tm.Id != managerId {
		tm.newInterval()
		req.Intervals = tm.getMissingIntervalsForProc(tm.Id, tm.getHighestTimestamp(managerId))
		tm.sendMessage(managerId, BARRREQ, req)
	} else {
		tm.barrierRequestHandler(req)
	}
	<-tm.channel
}

func (tm *TreadMarks) sendBarrierResponse(to uint8, ts Timestamp) {
	resp := BarrierResponse{
		Intervals: tm.getMissingIntervals(ts),
		Timestamp: tm.timestamp,
	}
	tm.sendMessage(to, BARRRSP, resp)
}

func (tm *TreadMarks) sendCopyRequest(pageNr int16) {
	page := tm.pageArray[pageNr]
	copySet := page.copySet
	to := copySet[len(copySet)-1] //yy: last one is cached in this processor
	if to != tm.Id {
		req := CopyRequest{
			From:   tm.Id,
			PageNr: pageNr,
		}
		tm.sendMessage(to, COPYREQ, req)
	} else {
		page.hasCopy = true
		tm.channel <- true
	}
}

func (tm *TreadMarks) sendCopyResponse(to uint8, pageNr int16) {
	data := make([]byte, tm.pageByteSize)
	tm.twinsLock.Lock()
	copy(data, tm.twins[pageNr])
	tm.twinsLock.Unlock()
	if data == nil {
		pageSize := tm.memory.PageSize()
		addr := int(pageNr) * pageSize
		copy(data, tm.memory.ForceRead(addr, pageSize))
	}
	resp := CopyResponse{
		PageNr: pageNr,
		Data:   data,
	}
	tm.sendMessage(to, COPYRSP, resp)
}

func (tm *TreadMarks) sendDiffRequests(pageNr int16) {
	diffRequests := tm.createDiffRequests(pageNr)
	for _, req := range diffRequests {
		tm.sendMessage(req.to, DIFFREQ, req)
	}
	for range diffRequests {
		<-tm.channel
	}
	tm.pageArray[pageNr].hasMissingDiffs = false
}

func (tm *TreadMarks) sendDiffResponse(to uint8, pageNr int16, writenotices []WritenoticeRecord) {
	resp := DiffResponse{
		PageNr:       pageNr,
		Writenotices: writenotices,
	}
	tm.sendMessage(to, DIFFRSP, resp)
}

func (tm *TreadMarks) managerID(id uint8) uint8 {
	return 0
}

func (tm *TreadMarks) getHighestTimestamp(procId uint8) Timestamp {
	if len(tm.procArray[procId]) == 0 {
		return NewTimestamp(tm.numProcs)
	}
	ts := tm.procArray[procId][len(tm.procArray[procId])-1].Timestamp
	return ts
}

///////////////////////////////
// incoming message handlers
///////////////////////////////


func (tm *TreadMarks) messageHandler() {
	tm.group.Add(1)
Loop:
	for {
		time.Sleep(0)
		var msg []byte
		select {
		case msg = <-tm.in:
		case <-tm.shutdown:
			break Loop
		}
		buf := msg[2:]
		event := "Receive " + msgName[msg[1]]
		opt:= govec.GetDefaultLogOptions()
		switch msg[1] {
		case LOCKREQ: //lock acquire request
			var req LockAcquireRequest
			tm.vecLog.UnpackReceive(event, buf, &req, opt)
			tm.lockAcquireRequestHandler(req)
		case LOCKRSP: //lock acquire response
			var resp LockAcquireResponse
			tm.vecLog.UnpackReceive(event, buf, &resp, opt)
			tm.lockAcquireResponseHandler(resp)
		case BARRREQ: //Barrier Request
			var req BarrierRequest
			tm.vecLog.UnpackReceive(event, buf, &req, opt)
			tm.barrierRequestHandler(req)
		case BARRRSP: //Barrier response
			var resp BarrierResponse
			tm.vecLog.UnpackReceive(event, buf, &resp, opt)
			tm.barrierResponseHandler(resp)
		case COPYREQ: //Copy request
			var req CopyRequest
			tm.vecLog.UnpackReceive(event, buf, &req, opt)
			tm.copyRequestHandler(req)
		case COPYRSP: //Copy response
			var resp CopyResponse
			tm.vecLog.UnpackReceive(event, buf, &resp, opt)
			tm.copyResponseHandler(resp)
		case DIFFREQ: // Diff request
			var req DiffRequest
			req.From = uint8(msg[0])
			tm.vecLog.UnpackReceive(event, buf, &req, opt)
			tm.diffRequestHandler(req)
		case DIFFRSP: //Diff response
			var resp DiffResponse
			tm.vecLog.UnpackReceive(event, buf, &resp, opt)
			tm.diffResponseHandler(resp)
		}
	}
	tm.group.Done()
}

func (tm *TreadMarks) lockAcquireRequestHandler(req LockAcquireRequest) {
	id := req.LockId
	lock := tm.locks[id]
	lock.Lock()
	// using lock right now, will forward the lock when releasing the lock
	if lock.locked {
		if lock.nextTimestamp != nil {
			tm.forwardLockAcquireRequest(lock.last, req)
		} else {
			lock.nextId = req.From
			lock.nextTimestamp = req.Timestamp
		}
		lock.last = req.From
	} else {
		if lock.haveToken {
			lock.haveToken = false
			lock.last = req.From
			tm.newInterval()
			tm.sendLockAcquireResponse(id, req.From, req.Timestamp)
		} else {
		// lock will come back in the future, just wait here	
			if lock.last != tm.Id { 
				tm.forwardLockAcquireRequest(lock.last, req)
			} else {
				if lock.nextTimestamp != nil {
					// forward request and wait in line
					tm.forwardLockAcquireRequest(lock.last, req)
				} else {
					lock.nextTimestamp = req.Timestamp
					lock.nextId = req.From
				}
			} 
			lock.last = req.From
		}
	}
	lock.Unlock()
}

func (tm *TreadMarks) lockAcquireResponseHandler(resp LockAcquireResponse) {
	id := resp.LockId
	lock := tm.locks[id]
	lock.Lock()
	tm.newInterval()
	for i := len(resp.Intervals); i > 0; i-- {
		tm.addInterval(resp.Intervals[i-1])
	}
	lock.locked = true
	lock.haveToken = true
	lock.Unlock()
	tm.channel <- true
}

func (tm *TreadMarks) barrierRequestHandler(req BarrierRequest) {
	tm.barrierreq[req.From] = req
	n := <-tm.barrier + 1
	if n < tm.numProcs {
		tm.barrier <- n
	} else {
		tm.newInterval()
		for _, req := range tm.barrierreq {
			for i := len(req.Intervals); i > 0; i-- {
				tm.addInterval(req.Intervals[i-1])
			}
		}
		
		for i := uint8(0); i < tm.numProcs; i++ {
			if i != tm.Id {
				tm.sendBarrierResponse(i, tm.barrierreq[i].Timestamp)
			}
		}
		tm.barrier <- 0
		tm.channel <- true
	}
}

func (tm *TreadMarks) barrierResponseHandler(resp BarrierResponse) {
	for i := len(resp.Intervals); i > 0; i-- {
		tm.addInterval(resp.Intervals[i-1])
	}
	tm.channel <- true
}

func (tm *TreadMarks) copyRequestHandler(req CopyRequest) {
	tm.sendCopyResponse(req.From, req.PageNr)
	copyset := tm.pageArray[req.PageNr].copySet
	copyset = append(copyset, req.From)
}

func (tm *TreadMarks) copyResponseHandler(resp CopyResponse) {
	tm.memory.ForceWrite(int(resp.PageNr)*tm.memory.PageSize(), resp.Data)
	page := tm.pageArray[resp.PageNr]
	page.hasCopy = true
	page.copySet = append(page.copySet, tm.Id)
	tm.channel <- true
}

func (tm *TreadMarks) diffRequestHandler(req DiffRequest) {
	tm.twinsLock.Lock()
	if tm.twins[req.PageNr] != nil {
		tm.dirtyPagesLock.Lock()
		if tm.dirtyPages[req.PageNr] {
			tm.newWritenoticeRecord(req.PageNr)
		}
		tm.dirtyPagesLock.Unlock()
		tm.generateDiff(req.PageNr, tm.twins[req.PageNr])
		tm.twins[req.PageNr] = nil
	}

	result := make([]WritenoticeRecord, 0)
	var proc uint8
	for proc = 0; proc < tm.numProcs; proc++ {
		if proc == req.From {
			continue
		}
		list := tm.pageArray[req.PageNr].writenotices[proc]
		for i := len(list) - 1; i >= 0; i-- {
			wn := list[i]
			if req.Last.larger(wn.Timestamp) {
				result = append(result, wn)
			}
			if !wn.Timestamp.larger(req.First) {
				break
			} 
		}
	}
	tm.sendDiffResponse(req.From, req.PageNr, result)
	tm.twinsLock.Unlock()
}

func (tm *TreadMarks) diffResponseHandler(resp DiffResponse) {
	wnl := resp.Writenotices
	j := 0
	for proc := uint8(0); proc < tm.numProcs; proc++ {
		if j >= len(wnl) {
			break
		}
		if proc == tm.Id {
			continue
		}

		list := tm.pageArray[resp.PageNr].writenotices[proc]
		for i := len(list) - 1; i >= 0; i-- {
			j++
			if !(len(wnl) > j) {
				break
			}
			if !list[i].Timestamp.equals(wnl[j].Timestamp) {
				break
			}
			list[i].Diff = wnl[j].Diff
		}
	}
	tm.channel <- true
}

func (tm *TreadMarks) putDiffs(pageNr int16) {

	x := 0
	tm.diffLock.Lock()
	defer tm.diffLock.Unlock()
	wnl := tm.pageArray[pageNr].writenotices
	// curPos stores the current writenotice's position
	curPos := tm.pageArray[pageNr].curPos
	for {
		var bestTs Timestamp = nil
		var best uint8 = 0
		var proc uint8

		for proc = 0; proc < tm.numProcs; proc++ {
			if curPos[proc] < len(wnl[proc]) {
				wn := wnl[proc][curPos[proc]]
				if bestTs == nil || !wn.Timestamp.larger(bestTs) { 
					best = proc
					bestTs = wn.Timestamp
				}
			}
		}
		if bestTs == nil {
			break
		}
		
		curPos[best] = curPos[best] + 1
		x++
		diff := wnl[best][curPos[best]].Diff
		tm.putDiff(pageNr, diff)

	}

	tm.pageArray[pageNr].curPos = curPos
}

func (tm *TreadMarks) putDiff(pageNr int16, diff map[int]byte) {

	size := tm.memory.PageSize()
	addr := int(pageNr) * size
	data := tm.memory.ForceRead(addr, size)

	for key, value := range diff {
		data[key] = value
	}
	tm.memory.ForceWrite(addr, data)
}



func (tm *TreadMarks) GetId() int {
	return int(tm.Id)
}

func (tm *TreadMarks) log(msgId uint8) {
	tm.messageLog[msgId]++
}
