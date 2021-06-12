package src

import (
	"fmt"
	"log"
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

func valueEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		log.Println(a, " ", b)
		debug.PrintStack()
		t.Fatal()
	}
}
func nilEqual(t *testing.T, err interface{}) {
	if err != nil {
		log.Println(err)
		debug.PrintStack()
		t.Fatal()
	}
}
func nilUnequal(t *testing.T, err interface{}) {
	if err == nil {
		debug.PrintStack()
		t.Fatal()
	}
}
func trueEqual(t *testing.T, b bool) {
	if b != true {
		debug.PrintStack()
		t.Fatal()
	}
}
func falseEqual(t *testing.T, b bool) {
	if b != false {
		debug.PrintStack()
		t.Fatal()
	}
}

func sliceEqual(t *testing.T, a []int, b []int) {
	if len(a) != len(b) {
		debug.PrintStack()
		t.Fatal()
	}
	for i := range a {
		if a[i] != b[i] {
			debug.PrintStack()
			t.Fatal()
		}
	}
}
func TestNewTreadmarksApi_ReadWriteSingleProc(t *testing.T) {

	tm, err := NewTreadMarks(1024, 128, 1, 1, 1)

	nilEqual(t, err)
	err = tm.Initialize(1000)
	nilEqual(t, err)
	err = tm.Write(1, byte(2))

	nilEqual(t, err)
	data, err := tm.Read(1)
	nilEqual(t, err)
	valueEqual(t, byte(2), data)
	data, err = tm.Read(200)
	nilEqual(t, err)
	valueEqual(t, byte(0), data)
	tm.Shutdown()
}

func TestNewTreadmarksApi_LockAcquireReleaseSingleProc(t *testing.T) {

	tm, err := NewTreadMarks(1024, 128, 1, 1, 1)
	nilEqual(t, err)
	err = tm.Initialize(1000)
	trueEqual(t, tm.locks[0].haveToken)
	falseEqual(t, tm.locks[0].locked)
	tm.AcquireLock(0)
	trueEqual(t, tm.locks[0].haveToken)
	trueEqual(t, tm.locks[0].locked)
	tm.ReleaseLock(0)
	trueEqual(t, tm.locks[0].haveToken)
	falseEqual(t, tm.locks[0].locked)
	tm.Shutdown()
}

func TestNewTreadmarksApi_LockAcquireReleaseMultipleProc_sequential(t *testing.T) {
	tm1, err := NewTreadMarks(1024, 128, 2, 1, 1)
	nilEqual(t, err)
	err = tm1.Initialize(1000)
	nilEqual(t, err)
	tm2, err := NewTreadMarks(1024, 128, 2, 1, 1)
	nilEqual(t, err)
	err = tm2.Initialize(1001)
	nilEqual(t, err)
	err = tm2.Join("localhost", 1000)
	valueEqual(t, uint8(1), tm2.Id)
	nilEqual(t, err)
	trueEqual(t, tm1.locks[0].haveToken)
	falseEqual(t, tm1.locks[0].locked)
	falseEqual(t, tm2.locks[0].haveToken)
	falseEqual(t, tm2.locks[0].locked)
	fmt.Println("First acquire")
	tm1.AcquireLock(0)
	fmt.Println("First acquire passed")
	trueEqual(t, tm1.locks[0].haveToken)
	trueEqual(t, tm1.locks[0].locked)
	falseEqual(t, tm2.locks[0].haveToken)
	falseEqual(t, tm2.locks[0].locked)
	fmt.Println("First release")
	tm1.ReleaseLock(0)
	fmt.Println("First release passed")
	trueEqual(t, tm1.locks[0].haveToken)
	falseEqual(t, tm1.locks[0].locked)
	falseEqual(t, tm2.locks[0].haveToken)
	falseEqual(t, tm2.locks[0].locked)
	fmt.Println("Second acquire")
	tm2.AcquireLock(0)
	fmt.Println("Second acquire passed")
	falseEqual(t, tm1.locks[0].haveToken)
	falseEqual(t, tm1.locks[0].locked)
	trueEqual(t, tm2.locks[0].haveToken)
	trueEqual(t, tm2.locks[0].locked)
	fmt.Println("Second release")
	tm2.ReleaseLock(0)
	fmt.Println("Second release passed")
	falseEqual(t, tm1.locks[0].haveToken)
	falseEqual(t, tm1.locks[0].locked)
	trueEqual(t, tm2.locks[0].haveToken)
	falseEqual(t, tm2.locks[0].locked)
	fmt.Println("Third acquire")
	tm1.AcquireLock(0)
	fmt.Println("Third acquire passed")
	trueEqual(t, tm1.locks[0].haveToken)
	trueEqual(t, tm1.locks[0].locked)
	falseEqual(t, tm2.locks[0].haveToken)
	falseEqual(t, tm2.locks[0].locked)
	fmt.Println("Third release")
	tm1.ReleaseLock(0)
	fmt.Println("Third release passed")
	trueEqual(t, tm1.locks[0].haveToken)
	falseEqual(t, tm1.locks[0].locked)
	falseEqual(t, tm2.locks[0].haveToken)
	falseEqual(t, tm2.locks[0].locked)
	tm1.Shutdown()
	tm2.Shutdown()
}

func TestNewTreadmarksApi_BarrierMultipleProc_sequential(t *testing.T) {
	tm1, err := NewTreadMarks(1024, 128, 2, 1, 1)
	nilEqual(t, err)
	err = tm1.Initialize(1000)
	defer tm1.Shutdown()
	nilEqual(t, err)
	tm2, err := NewTreadMarks(1024, 128, 2, 1, 1)
	nilEqual(t, err)
	err = tm2.Initialize(1001)
	defer tm2.Shutdown()
	nilEqual(t, err)
	err = tm2.Join("localhost", 1000)
	valueEqual(t, uint8(1), tm2.Id)
	nilEqual(t, err)
	done := false
	go func() {
		tm1.Barrier(0)
		done = true
	}()
	falseEqual(t, done)
	tm2.Barrier(0)
	time.Sleep(time.Millisecond * 100)
	trueEqual(t, done)

}

func TestNewTreadmarksApi_LockAcquireReleaseMultipleProc_Concurrent(t *testing.T) {

	tm0, _ := NewTreadMarks(1024, 128, 3, 3, 1)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadMarks(1024, 128, 3, 3, 1)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadMarks(1024, 128, 3, 3, 1)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

	go0, go1, go2 := make(chan bool), make(chan bool), make(chan bool)
	done := make([]int, 0, 100)
	go func() {
		tm := tm0
		g := go0
		<-g
		fmt.Println(tm.Id, " beginning to acquire lock.")
		tm.AcquireLock(1)
		fmt.Println(tm.Id, " Lock acquired")
		done = append(done, 0)
		g <- true
		<-g
		fmt.Println(tm.Id, " beginning to release lock.")
		tm.ReleaseLock(1)
		fmt.Println(tm.Id, " Lock released")
		done = append(done, 0)
	}()
	go func() {
		tm := tm1
		g := go1
		<-g
		fmt.Println(tm.Id, " beginning to acquire lock.")
		tm.AcquireLock(1)
		fmt.Println(tm.Id, " Lock acquired")
		done = append(done, 1)
		g <- true
		<-g
		fmt.Println(tm.Id, " beginning to release lock.")
		tm.ReleaseLock(1)
		fmt.Println(tm.Id, " Lock released")
		done = append(done, 1)
	}()
	go func() {
		tm := tm2
		g := go2
		<-g
		fmt.Println(tm.Id, " beginning to acquire lock.")
		tm.AcquireLock(1)
		fmt.Println(tm.Id, " Lock acquired")
		done = append(done, 2)
		g <- true
		<-g
		fmt.Println(tm.Id, " beginning to release lock.")
		tm.ReleaseLock(1)
		fmt.Println(tm.Id, " Lock released")
		done = append(done, 2)
	}()
	go0 <- true
	time.Sleep(time.Millisecond * 10)
	go1 <- true
	time.Sleep(time.Millisecond * 10)
	go2 <- true
	time.Sleep(time.Millisecond * 10)
	<-go0
	valueEqual(t, len(done), 1)
	valueEqual(t, 0, done[0])

	go0 <- true
	<-go1
	time.Sleep(time.Millisecond * 10)
	valueEqual(t, len(done), 3)
	sliceEqual(t, []int{0, 0, 1}, done)

	go1 <- true
	<-go2
	time.Sleep(time.Millisecond * 10)
	valueEqual(t, len(done), 5)
	sliceEqual(t, []int{0, 0, 1, 1, 2}, done)
	go2 <- true
}

func TestTreadmarksApi_Barrier_MultiplePeers(t *testing.T) {

	tm1, _ := NewTreadMarks(1024, 128, 3, 3, 3)
	tm1.Initialize(1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadMarks(1024, 128, 3, 3, 3)
	tm2.Initialize(1001)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()
	tm3, _ := NewTreadMarks(1024, 128, 3, 3, 3)
	tm3.Initialize(1002)
	tm3.Join("localhost", 1000)
	defer tm3.Shutdown()

	go1, go2, go3 := make(chan bool), make(chan bool), make(chan bool)

	go func() {
		<-go1
		fmt.Println("0 going to barrier")
		tm1.Barrier(1)
		fmt.Println("0 Done")
		go1 <- true
	}()
	go func() {
		<-go2
		fmt.Println("1 going to barrier")
		tm2.Barrier(1)
		fmt.Println("1 Done")
		go2 <- true
	}()
	go func() {
		<-go3
		fmt.Println("2 going to barrier")
		tm3.Barrier(1)
		fmt.Println("2 Done")
		go3 <- true
	}()
	go1 <- true
	go2 <- true
	trueEqual(t, timeout(go1))
	trueEqual(t, timeout(go2))
	trueEqual(t, timeout(go3))

	go3 <- true

	falseEqual(t, timeout(go1))
	falseEqual(t, timeout(go2))
	falseEqual(t, timeout(go3))
}

func TestNewTreadmarksApi_ReadAndWriteSingle(t *testing.T) {

	tm1, _ := NewTreadMarks(1024, 128, 1, 3, 3)
	tm1.Initialize(1000)
	defer tm1.Shutdown()

	go1 := make(chan bool)
	go func() {
		data, err := tm1.Read(1)
		valueEqual(t, data, byte(0))
		nilEqual(t, err)
		err = tm1.Write(1, byte(2))
		nilEqual(t, err)
		data, err = tm1.Read(1)
		valueEqual(t, data, byte(2))
		nilEqual(t, err)
		go1 <- true
	}()

	trueEqual(t, <-go1)
}

func TestNewTreadmarksApi_ReadAndWriteSingleOther(t *testing.T) {
	tm1, _ := NewTreadMarks(1024, 128, 3, 3, 3)
	tm1.Initialize(1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadMarks(1024, 128, 3, 3, 3)
	tm2.Initialize(1001)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

	go2 := make(chan bool)
	go func() {
		data, err := tm2.Read(1)
		valueEqual(t, data, byte(0))
		nilEqual(t, err)
		err = tm2.Write(1, byte(2))
		nilEqual(t, err)
		data, err = tm2.Read(1)
		valueEqual(t, data, byte(2))
		nilEqual(t, err)
		go2 <- true
	}()
	trueEqual(t, <-go2)
}

func TestNewTreadmarksApi_ReadAndWriteMultiple(t *testing.T) {
	tm0, _ := NewTreadMarks(1024, 128, 2, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadMarks(1024, 128, 2, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()

	go1, go2 := make(chan bool), make(chan bool)
	go func() {
		data, err := tm0.Read(1)
		valueEqual(t, data, byte(0))
		nilEqual(t, err)
		err = tm0.Write(1, byte(1))
		nilEqual(t, err)
		data, err = tm0.Read(1)
		valueEqual(t, data, byte(1))
		nilEqual(t, err)
		fmt.Println("0 hit barrier")
		tm0.Barrier(1)
		fmt.Println("0 passed barrier")
		go1 <- true
	}()
	go func() {
		fmt.Println("1 hit barrier")
		tm1.Barrier(1)
		fmt.Println("1 passed barrier")
		data, err := tm1.Read(1)
		valueEqual(t, data, byte(1))
		nilEqual(t, err)
		go2 <- true
	}()
	trueEqual(t, <-go1)
	trueEqual(t, <-go2)
}

func TestNewTreadmarksApi_ReadAndWriteMultiple2(t *testing.T) {
	tm0, _ := NewTreadMarks(1024, 128, 3, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadMarks(1024, 128, 3, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadMarks(1024, 128, 3, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

	done := make(chan bool, 2)

	go func() {
		tm := tm1
		tm.Write(1, byte(2))
		time.Sleep(time.Millisecond * 50)
		tm.AcquireLock(1)
		tm.ReleaseLock(1)
		b, _ := tm.Read(1)
		valueEqual(t, byte(2), b)
		time.Sleep(time.Millisecond * 100)
		tm.AcquireLock(1)
		tm.ReleaseLock(1)
		fmt.Println("lock released")
		b, _ = tm.Read(1)
		valueEqual(t, byte(2), b)
		done <- true
	}()
	go func() {
		tm := tm2
		tm.Write(1, byte(4))
		time.Sleep(time.Millisecond * 100)
		tm.AcquireLock(1)
		tm.ReleaseLock(1)
		b, _ := tm.Read(1)
		valueEqual(t, byte(2), b)
		done <- true
	}()
	<-done
	<-done
}

func TestNewTreadmarksApi_ReadAndWriteMultiple3(t *testing.T) {
	runtime.GOMAXPROCS(2)
	tm0, _ := NewTreadMarks(1024*4, 128, 2, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadMarks(1024*4, 128, 2, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	done := make(chan bool, 2)
	go func() {
		tm := tm0
		tm.Write(0, 1)
		fmt.Println(tm.Id, " at barrier 0")
		tm.Barrier(0)
		fmt.Println(tm.Id, " past barrier 0")
		for {
			tm.AcquireLock(0)
			n, _ := tm.Read(0)
			if !(n < 200) {
				tm.ReleaseLock(0)
				break
			}
			tm.Write(0, n+10)
			tm.ReleaseLock(0)
			var k uint8
			for k = n; k < n+10 && k < 200; k++ {
				tm.Write(int(k), k)
			}
			fmt.Println(tm.Id, " done iterating n=", k)
		}
		fmt.Println(tm.Id, "at barrier 1")
		tm.Barrier(1)

		done <- true
	}()
	go func() {
		tm := tm1
		fmt.Println(tm.Id, " at barrier 0")
		tm.Barrier(0)
		fmt.Println(tm.Id, " past barrier 0")
		for {
			tm.AcquireLock(0)
			n, _ := tm.Read(0)
			if !(n < 200) {
				tm.ReleaseLock(0)
				break
			}
			tm.Write(0, n+10)
			tm.ReleaseLock(0)
			var k uint8
			for k = n; k < n+10 && k < 200; k++ {
				tm.Write(int(k), k)
			}
			fmt.Println(tm.Id, " done iterating n=", k)
		}
		fmt.Println(tm.Id, "at barrier 1")
		tm.Barrier(1)

		done <- true
	}()
	<-done
	<-done
	var n uint8
	for n = 1; n < 200; n++ {
		val, _ := tm0.Read(int(n))
		valueEqual(t, n, val)
	}
	for n = 1; n < 200; n++ {
		val, _ := tm1.Read(int(n))
		valueEqual(t, n, val)
	}
	fmt.Println(tm0.procarray[0])
	fmt.Println(tm1.procarray[0])
	fmt.Println(tm0.procarray[1])
	fmt.Println(tm1.procarray[1])
}

func TestNewTreadmarksApi_locks(t *testing.T) {
	tm0, _ := NewTreadMarks(1024, 128, 3, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadMarks(1024, 128, 3, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadMarks(1024, 128, 3, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

	go2 := make(chan bool, 1)

	var lockId uint8 = 0

	lock0, lock1, lock2 := tm0.locks[lockId], tm1.locks[lockId], tm2.locks[lockId]

	falseEqual(t, lock0.locked)
	falseEqual(t, lock1.locked)
	falseEqual(t, lock2.locked)
	trueEqual(t, lock0.haveToken)
	falseEqual(t, lock1.haveToken)
	falseEqual(t, lock2.haveToken)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	valueEqual(t, len(lock1.nextTimestamp), 0)
	valueEqual(t, len(lock2.nextTimestamp), 0)
	valueEqual(t, tm0.getManagerId(lockId), lock0.last)
	valueEqual(t, tm1.getManagerId(lockId), lock1.last)
	valueEqual(t, tm2.getManagerId(lockId), lock2.last)

	tm0.AcquireLock(lockId)

	trueEqual(t, lock0.locked)
	falseEqual(t, lock1.locked)
	falseEqual(t, lock2.locked)
	trueEqual(t, lock0.haveToken)
	falseEqual(t, lock1.haveToken)
	falseEqual(t, lock2.haveToken)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	valueEqual(t, len(lock1.nextTimestamp), 0)
	valueEqual(t, len(lock2.nextTimestamp), 0)
	valueEqual(t, tm0.getManagerId(lockId), lock0.last)
	valueEqual(t, tm1.getManagerId(lockId), lock1.last)
	valueEqual(t, tm2.getManagerId(lockId), lock2.last)

	tm0.ReleaseLock(lockId)

	falseEqual(t, lock0.locked)
	falseEqual(t, lock1.locked)
	falseEqual(t, lock2.locked)
	trueEqual(t, lock0.haveToken)
	falseEqual(t, lock1.haveToken)
	falseEqual(t, lock2.haveToken)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	valueEqual(t, len(lock1.nextTimestamp), 0)
	valueEqual(t, len(lock2.nextTimestamp), 0)
	valueEqual(t, tm0.getManagerId(lockId), lock0.last)
	valueEqual(t, tm1.getManagerId(lockId), lock1.last)
	valueEqual(t, tm2.getManagerId(lockId), lock2.last)

	tm1.AcquireLock(lockId)

	falseEqual(t, lock0.locked)
	trueEqual(t, lock1.locked)
	falseEqual(t, lock2.locked)
	falseEqual(t, lock0.haveToken)
	trueEqual(t, lock1.haveToken)
	falseEqual(t, lock2.haveToken)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	valueEqual(t, len(lock1.nextTimestamp), 0)
	valueEqual(t, len(lock2.nextTimestamp), 0)
	valueEqual(t, uint8(1), lock0.last)
	valueEqual(t, tm1.Id, lock1.last)
	valueEqual(t, tm2.getManagerId(lockId), lock2.last)

	go func() {
		go2 <- true
		tm2.AcquireLock(lockId)
		go2 <- true

	}()
	<-go2
	time.Sleep(time.Millisecond * 100)
	falseEqual(t, lock0.locked)
	trueEqual(t, lock1.locked)
	falseEqual(t, lock2.locked)
	falseEqual(t, lock0.haveToken)
	trueEqual(t, lock1.haveToken)
	falseEqual(t, lock2.haveToken)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	nilUnequal(t, lock1.nextTimestamp)
	valueEqual(t, len(lock2.nextTimestamp), 0)
	valueEqual(t, uint8(2), lock0.last)
	valueEqual(t, uint8(2), lock1.nextId)
	valueEqual(t, tm2.getManagerId(lockId), lock2.nextId)
	tm1.ReleaseLock(lockId)
	<-go2
	time.Sleep(time.Millisecond * 100)
	falseEqual(t, lock0.locked)
	falseEqual(t, lock1.locked)
	trueEqual(t, lock2.locked)
	falseEqual(t, lock0.haveToken)
	falseEqual(t, lock1.haveToken)
	trueEqual(t, lock2.haveToken)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	valueEqual(t, len(lock1.nextTimestamp), 0)
	valueEqual(t, len(lock2.nextTimestamp), 0)
	valueEqual(t, uint8(2), lock0.last)
	//yy : not sure about this
	//valueEqual(t, tm1.getManagerId(lockId), lock1.last)
	valueEqual(t, uint8(2), lock1.last)
	valueEqual(t, tm2.Id, lock2.last)
	valueEqual(t, tm1.getManagerId(lockId), lock1.nextId)
	valueEqual(t, tm2.getManagerId(lockId), lock2.nextId)
}

func TestTreadmarksApiTestLock(t *testing.T) {
	tm0, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()
	tm3, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm3.Initialize(1003)
	tm3.Join("localhost", 1000)
	defer tm3.Shutdown()

	//go0:= make(chan bool, 1)
	//go1 := make(chan bool, 1)
	//go2 :=  make(chan bool, 1)
	//go3 :=  make(chan bool, 1)

	var lockId uint8 = 0

	lock0, lock1, lock2, lock3 := tm0.locks[lockId], tm1.locks[lockId], tm2.locks[lockId], tm3.locks[lockId]

	go func() {
		tm1.AcquireLock(lockId)
	}()
	time.Sleep(time.Millisecond * 100)
	go func() {
		tm2.AcquireLock(lockId)
	}()
	time.Sleep(time.Millisecond * 100)
	go func() {
		tm3.AcquireLock(lockId)
	}()
	time.Sleep(time.Millisecond * 100)

	falseEqual(t, lock0.locked)
	trueEqual(t, lock1.locked)
	falseEqual(t, lock2.locked)
	falseEqual(t, lock3.locked)
	falseEqual(t, lock0.haveToken)
	trueEqual(t, lock1.haveToken)
	falseEqual(t, lock2.haveToken)
	falseEqual(t, lock3.haveToken)
	valueEqual(t, byte(3), lock0.last)
	valueEqual(t, byte(2), lock1.nextId)
	valueEqual(t, byte(3), lock2.nextId)
	valueEqual(t, byte(0), lock3.nextId)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	nilUnequal(t, lock1.nextTimestamp)
	nilUnequal(t, lock2.nextTimestamp)
	valueEqual(t, len(lock3.nextTimestamp), 0)

	tm1.ReleaseLock(lockId)

	time.Sleep(time.Millisecond * 100)

	falseEqual(t, lock0.locked)
	falseEqual(t, lock1.locked)
	trueEqual(t, lock2.locked)
	falseEqual(t, lock3.locked)
	falseEqual(t, lock0.haveToken)
	falseEqual(t, lock1.haveToken)
	trueEqual(t, lock2.haveToken)
	falseEqual(t, lock3.haveToken)
	valueEqual(t, byte(3), lock0.last)
	valueEqual(t, byte(0), lock1.nextId)
	valueEqual(t, byte(3), lock2.nextId)
	valueEqual(t, byte(0), lock3.nextId)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	valueEqual(t, len(lock1.nextTimestamp), 0)
	nilUnequal(t, lock2.nextTimestamp)
	valueEqual(t, len(lock3.nextTimestamp), 0)

	tm2.ReleaseLock(lockId)

	time.Sleep(time.Millisecond * 100)

	falseEqual(t, lock0.locked)
	falseEqual(t, lock1.locked)
	falseEqual(t, lock2.locked)
	trueEqual(t, lock3.locked)
	falseEqual(t, lock0.haveToken)
	falseEqual(t, lock1.haveToken)
	falseEqual(t, lock2.haveToken)
	trueEqual(t, lock3.haveToken)
	valueEqual(t, byte(3), lock0.last)
	valueEqual(t, byte(0), lock1.nextId)
	valueEqual(t, byte(0), lock2.nextId)
	valueEqual(t, byte(0), lock3.nextId)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	valueEqual(t, len(lock1.nextTimestamp), 0)
	valueEqual(t, len(lock2.nextTimestamp), 0)
	valueEqual(t, len(lock3.nextTimestamp), 0)
}

func TestTreadmarksApiTestLock2(t *testing.T) {
	tm0, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm0.Initialize(1000)
	tm1, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	tm2, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	tm3, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm3.Initialize(1003)
	tm3.Join("localhost", 1000)
	defer tm0.Shutdown()
	defer tm1.Shutdown()
	defer tm2.Shutdown()
	defer tm3.Shutdown()


	var lockId uint8 = 0

	lock0, lock1, lock2, lock3 := tm0.locks[lockId], tm1.locks[lockId], tm2.locks[lockId], tm3.locks[lockId]

	tm1.AcquireLock(lockId)
	tm1.ReleaseLock(lockId)
	time.Sleep(time.Millisecond * 100)

	falseEqual(t, lock0.locked)
	falseEqual(t, lock1.locked)
	falseEqual(t, lock2.locked)
	falseEqual(t, lock3.locked)
	falseEqual(t, lock0.haveToken)
	trueEqual(t, lock1.haveToken)
	falseEqual(t, lock2.haveToken)
	falseEqual(t, lock3.haveToken)
	valueEqual(t, byte(1), lock0.last)
	valueEqual(t, byte(0), lock1.nextId)
	valueEqual(t, byte(0), lock2.nextId)
	valueEqual(t, byte(0), lock3.nextId)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	valueEqual(t, len(lock1.nextTimestamp), 0)
	valueEqual(t, len(lock2.nextTimestamp), 0)
	valueEqual(t, len(lock3.nextTimestamp), 0)

	tm2.AcquireLock(lockId)
	tm2.ReleaseLock(lockId)

	time.Sleep(time.Millisecond * 100)

	falseEqual(t, lock0.locked)
	falseEqual(t, lock1.locked)
	falseEqual(t, lock2.locked)
	falseEqual(t, lock3.locked)
	falseEqual(t, lock0.haveToken)
	falseEqual(t, lock1.haveToken)
	trueEqual(t, lock2.haveToken)
	falseEqual(t, lock3.haveToken)
	valueEqual(t, byte(2), lock0.last)
	valueEqual(t, byte(0), lock1.nextId)
	valueEqual(t, byte(0), lock2.nextId)
	valueEqual(t, byte(0), lock3.nextId)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	valueEqual(t, len(lock1.nextTimestamp), 0)
	valueEqual(t, len(lock2.nextTimestamp), 0)
	valueEqual(t, len(lock3.nextTimestamp), 0)

	tm3.AcquireLock(lockId)
	tm3.ReleaseLock(lockId)

	time.Sleep(time.Millisecond * 100)

	falseEqual(t, lock0.locked)
	falseEqual(t, lock1.locked)
	falseEqual(t, lock2.locked)
	falseEqual(t, lock3.locked)
	falseEqual(t, lock0.haveToken)
	falseEqual(t, lock1.haveToken)
	falseEqual(t, lock2.haveToken)
	trueEqual(t, lock3.haveToken)
	valueEqual(t, byte(3), lock0.last)
	valueEqual(t, byte(0), lock1.nextId)
	valueEqual(t, byte(0), lock2.nextId)
	valueEqual(t, byte(0), lock3.nextId)
	valueEqual(t, len(lock0.nextTimestamp), 0)
	valueEqual(t, len(lock1.nextTimestamp), 0)
	valueEqual(t, len(lock2.nextTimestamp), 0)
	valueEqual(t, len(lock3.nextTimestamp), 0)
}

func TestTreadmarksApi_Barriers(t *testing.T) {
	tm0, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm0.Initialize(1000)
	tm1, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	tm2, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	tm3, _ := NewTreadMarks(1024, 128, 4, 3, 3)
	tm3.Initialize(1003)
	tm3.Join("localhost", 1000)

	defer tm0.Shutdown()
	defer tm1.Shutdown()
	defer tm2.Shutdown()
	defer tm3.Shutdown()

	go0 := make(chan bool)
	go1 := make(chan bool)
	go2 := make(chan bool)
	go3 := make(chan bool)

	test := func(tm *TreadmarksApi, g chan bool) {
		go func() {
			tm.Write(0, byte(0))
			tm.Barrier(0)
			g <- true
			<-g
			tm.Write(0, byte(0))
		}()
		time.Sleep(time.Millisecond * 100)
	}

	test(tm0, go0)
	test(tm1, go1)
	test(tm2, go2)

	trueEqual(t, timeout(go0))
	trueEqual(t, timeout(go1))
	trueEqual(t, timeout(go2))

	test(tm3, go3)

	falseEqual(t, timeout(go0))
	falseEqual(t, timeout(go1))
	falseEqual(t, timeout(go2))
	falseEqual(t, timeout(go3))
	go0 <- true
	go1 <- true
	go2 <- true
	go3 <- true
	time.Sleep(time.Millisecond * 100)

	test(tm1, go1)
	test(tm0, go0)
	test(tm2, go2)

	trueEqual(t, timeout(go1))
	trueEqual(t, timeout(go0))
	trueEqual(t, timeout(go2))

	test(tm3, go3)

	falseEqual(t, timeout(go0))
	falseEqual(t, timeout(go1))
	falseEqual(t, timeout(go2))
	falseEqual(t, timeout(go3))
	go0 <- true
	go1 <- true
	go2 <- true
	go3 <- true
	time.Sleep(time.Millisecond * 100)

	test(tm1, go1)
	test(tm2, go2)
	test(tm0, go0)

	trueEqual(t, timeout(go1))
	trueEqual(t, timeout(go0))
	trueEqual(t, timeout(go2))

	test(tm3, go3)
	falseEqual(t, timeout(go0))
	falseEqual(t, timeout(go1))
	falseEqual(t, timeout(go2))
	falseEqual(t, timeout(go3))
	go0 <- true
	go1 <- true
	go2 <- true
	go3 <- true
	time.Sleep(time.Millisecond * 100)

	test(tm1, go1)
	test(tm2, go2)
	test(tm3, go3)

	trueEqual(t, timeout(go1))
	trueEqual(t, timeout(go0))
	trueEqual(t, timeout(go2))

	test(tm0, go0)
	falseEqual(t, timeout(go0))
	falseEqual(t, timeout(go1))
	falseEqual(t, timeout(go2))
	falseEqual(t, timeout(go3))

}

func TestNewTreadmarksApi_LockIntervalsMultipleProcs(t *testing.T) {
	tm0, _ := NewTreadMarks(30, 10, 3, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadMarks(30, 10, 3, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadMarks(30, 10, 3, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

	tm0.AcquireLock(0)
	tm0.Write(0, 4)
	tm0.ReleaseLock(0)
	tm1.AcquireLock(0)
	tm1.Write(0, 5)
	tm1.ReleaseLock(0)
	tm0.AcquireLock(0)
	tm0.Write(0, 6)
	tm0.ReleaseLock(0)
	tm1.AcquireLock(0)
	tm1.Write(0, 7)
	tm1.ReleaseLock(0)
	tm0.AcquireLock(0)
	tm0.ReleaseLock(0)
	tm2.AcquireLock(0)
	val, err := tm2.Read(0)
	valueEqual(t, byte(7), val)
	nilEqual(t, err)
	valueEqual(t, len(tm0.procarray[0]), 2)
	valueEqual(t, len(tm1.procarray[0]), 2)
	valueEqual(t, len(tm2.procarray[0]), 2)
	valueEqual(t, len(tm0.procarray[1]), 2)
	valueEqual(t, len(tm1.procarray[1]), 2)
	valueEqual(t, len(tm2.procarray[1]), 2)
	valueEqual(t, len(tm0.procarray[2]), 0)
	valueEqual(t, len(tm1.procarray[2]), 0)
	valueEqual(t, len(tm2.procarray[2]), 0)

	valueEqual(t, len(tm0.pagearray[0].writenotices[0]), 2)
	valueEqual(t, len(tm1.pagearray[0].writenotices[0]), 2)
	valueEqual(t, len(tm2.pagearray[0].writenotices[0]), 2)
	valueEqual(t, len(tm0.pagearray[0].writenotices[1]), 2)
	valueEqual(t, len(tm1.pagearray[0].writenotices[1]), 2)
	valueEqual(t, len(tm2.pagearray[0].writenotices[1]), 2)
	valueEqual(t, len(tm0.pagearray[0].writenotices[2]), 0)
	valueEqual(t, len(tm1.pagearray[0].writenotices[2]), 0)
	valueEqual(t, len(tm2.pagearray[0].writenotices[2]), 0)
}

func TestNewTreadmarksApi_LockIntervalsMultipleProcs_2(t *testing.T) {
	tm0, _ := NewTreadMarks(30, 10, 3, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadMarks(30, 10, 3, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadMarks(30, 10, 3, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

	tm0.AcquireLock(0)
	tm0.Write(1, 10)
	tm0.ReleaseLock(0)
	tm0.AcquireLock(0)
	tm0.ReleaseLock(0)
	tm1.AcquireLock(0)
	val, _ := tm1.Read(1)
	valueEqual(t, byte(10), val)
	tm0.Write(2, 11)
	tm1.ReleaseLock(0)
	tm0.AcquireLock(0)
	val, _ = tm0.Read(2)
	valueEqual(t, byte(11), val)
	tm0.ReleaseLock(0)
	tm1.AcquireLock(0)
	val, _ = tm1.Read(2)
	valueEqual(t, byte(11), val)
	tm1.ReleaseLock(0)
	tm0.Write(12, 10)
	tm0.Write(24, 11)
	tm0.AcquireLock(0)
	tm0.ReleaseLock(0)
	tm1.AcquireLock(0)
	val, _ = tm1.Read(12)
	valueEqual(t, byte(10), val)
	val, _ = tm1.Read(24)
	valueEqual(t, byte(11), val)
}

func TestTreadmarksApi_RepeatedLock(t *testing.T) {
	tm0, _ := NewTreadMarks(30, 10, 3, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadMarks(30, 10, 3, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	runtime.GOMAXPROCS(4)
	go0, go1 := make(chan int), make(chan int)
	i := 0
	go func() {
		tm := tm0
		g := go0
		<-g
		time.Sleep(time.Millisecond * 100)
		fmt.Println("0")
		for {

			tm.AcquireLock(0)

			val, _ := tm.Read(0)
			tm.Write(0, val+1%256)
			tm.ReleaseLock(0)
			if val > 253 {
				fmt.Println("0, ", i)
				i++
				if i > 1000 {
					g <- 1
					return
				}
			}
		}

	}()
	go func() {
		tm := tm1
		g := go1
		<-g
		time.Sleep(time.Millisecond * 100)
		fmt.Println("1")
		for {
			tm.AcquireLock(0)

			val, _ := tm.Read(0)
			tm.Write(0, val+1%256)
			tm.ReleaseLock(0)
			if val > 253 {
				fmt.Println("1, ", i)
				i++
				if i > 1000 {
					g <- 1
					return
				}
			}
		}

	}()
	go0 <- 1
	go1 <- 2
	<-go1
	<-go0

}

func readInt(tm TreadMarksAPI, addr int) int {
	bInt := make([]byte, 4)
	var err error
	for i := range bInt {
		bInt[i], err = tm.Read(addr + i)
		if err != nil {
			panic(err.Error())
		}
	}
	output := int(BytesToInt32(bInt))
	fmt.Println("read - ", addr, " - ", output, " - ", bInt)
	return output
}

func writeInt(tm TreadMarksAPI, addr, input int) {
	bInt := Int32ToBytes(int32(input))
	fmt.Println("write - ", addr, " - ", input, " - ", bInt)
	var err error
	for i := range bInt {
		err = tm.Write(addr+i, bInt[i])
		if err != nil {
			panic(err.Error())
		}
	}
}

func timeout(channel chan bool) bool {
	select {
	case <-channel:
		return false
	case <-time.After(time.Millisecond * 100):
		return true
	}
}
