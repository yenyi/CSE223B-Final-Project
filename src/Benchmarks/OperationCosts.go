package Benchmarks

import (
	"fmt"
	"runtime/pprof"
	"time"
	treadmarks "treadmarks"
)

func TestBarrierTimeTM(nrTimes, nrhosts int) {
	tm1, tms := setupTMHosts(nrhosts, 4096, 4096)

	var startTime time.Time
	startTime = time.Now()

	for i := 0; i < nrTimes; i++ {
		for _, mw := range tms {
			go mw.Barrier(0)
		}
		tm1.Barrier(0)
	}
	pprof.StopCPUProfile()

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	for _, tm := range tms {
		tm.Shutdown()
	}
	tm1.Shutdown()
}

func TestLockTM(nrTimes int) {
	tm1, tms := setupTMHosts(4, 4096, 4096)

	var startTime time.Time
	startTime = time.Now()
	
	for i := 0; i < nrTimes; i++ {
		//fmt.Println("about to acquire lock at host", i%7)
		tms[i%3].AcquireLock(0)
		//fmt.Println("acquired lock at host", i%7)
		tms[i%3].ReleaseLock(0)
		//fmt.Println("released lock at host", i%7)

	}
	pprof.StopCPUProfile()

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	for _, tm := range tms {
		tm.Shutdown()
	}
	tm1.Shutdown()
}

func TestSynchronizedReadsWritesTM(nrRounds int) {
	tm1, tms := setupTMHosts(4, 4096, 4096)
	addr, _ := tm1.Malloc(1)

	var startTime time.Time
	startTime = time.Now()

	tm1.AcquireLock(0)
	tm1.Write(addr, byte(1))
	tm1.ReleaseLock(0)

	for i := 0; i < nrRounds; i++ {
		if i%3 == 0 {
			tms[i%3].AcquireLock(0)
			tms[i%3].Read(addr)
			tms[i%3].ReleaseLock(0)
		} else {
			tms[i%3].AcquireLock(0)
			tms[i%3].Write(addr, byte(i))
			tms[i%3].ReleaseLock(0)
		}
		
	//	fmt.Println("current value:", int(data))
		
	//	tms[1].AcquireLock(0)
	//	tms[1].Write(addr, byte(2))
	//	tms[1].ReleaseLock(0)
	//	tms[0].AcquireLock(0)
	//	data, _ = tms[0].Read(addr)
	//	fmt.Println("current value:", int(data))
	//	tms[0].ReleaseLock(0)
	}
	pprof.StopCPUProfile()

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	for _, tm := range tms {
		tm.Shutdown()
	}
	tm1.Shutdown()
}

func TestNonSynchronizedReadWritesTM(nrRounds int) {
	tm1, _ := setupTMHosts(1, 4096, 4096)
	addr, _ := tm1.Malloc(1)
	tm1.Write(addr, byte(1))
	var startTime time.Time
	startTime = time.Now()

	for i := 0; i < nrRounds; i++ {
		tm1.Write(addr, byte(1))
		tm1.Read(addr)
	}

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	tm1.Shutdown()
}

func readOnAllTMHosts(addr int, tms []*treadmarks.TreadMarks) {
	for _, tm := range tms {
		tm.Read(addr)
	}
}

func setupTMHosts(nrHosts int, memSize, pageByteSize int) (manager *treadmarks.TreadMarks, mws []*treadmarks.TreadMarks) {
	manager, _ = treadmarks.NewTreadMarks(memSize, pageByteSize, uint8(nrHosts), uint8(nrHosts), uint8(nrHosts))
	manager.Initialize(2000)
	mws = make([]*treadmarks.TreadMarks, nrHosts-1)
	for i := range mws {
		mws[i], _ = treadmarks.NewTreadMarks(memSize, pageByteSize, uint8(nrHosts), uint8(nrHosts), uint8(nrHosts))
		mws[i].Initialize(2000 + i + 1)
		mws[i].Join("localhost", 2000)
	}
	return
}


