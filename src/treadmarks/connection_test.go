package src

import (
	"bytes"
	"runtime/debug"
	"testing"
	"time"
)

func TestConnection_iptransform(t *testing.T) {
	ip := []string{
		"192.168.1.13",
		"192.168.1.14",
		"192.168.1.255",
		"localhost",
		"[::1]",
	}
	ipExpected := []string{
		"192.168.1.13",
		"192.168.1.14",
		"192.168.1.255",
		"127.0.0.1",
		"127.0.0.1",
	}
	port := []int{
		2000,
		2001,
		21,
		11,
		22,
	}
	stringEqual := func(a string, b string) {
		if a != b {
			debug.PrintStack()
			t.Fatal()
		}
	}

	intEqual := func(a int, b int) {
		if a != b {
			debug.PrintStack()
			t.Fatal()
		}
	}
	intListEqual := func(a []int, b []int) {
		if len(a) != len(b) {
			debug.PrintStack()
			t.Fatal()
		}
		for i := range a {
			intEqual(a[i], b[i])
		}
	}

	stringListEqual := func(a []string, b []string) {
		if len(a) != len(b) {
			debug.PrintStack()
			t.Fatal()
		}
		for i := range a {
			stringEqual(a[i], b[i])
		}
	}
	var sumaddrb []byte
	for i := range ip {
		addrb := addrToBytes(ip[i], port[i])
		sumaddrb = append(sumaddrb, addrb...)
		ipc, portc, _ := addrFromBytes(addrb)
		stringEqual(ipExpected[i], ipc)
		intEqual(port[i], portc)
	}
	ipclist := make([]string, 0)
	portclist := make([]int, 0)
	i := 0
	for i < len(sumaddrb) {
		ipc, portc, k := addrFromBytes(sumaddrb[i:])
		ipclist = append(ipclist, ipc)
		portclist = append(portclist, portc)
		i += k
	}
	stringListEqual(ipExpected, ipclist)
	intListEqual(port, portclist)
}

func TestNewConnection(t *testing.T) {
	intEqual := func(a int, b int) {
		if a != b {
			debug.PrintStack()
			t.Fatal()
		}
	}
	nilEqual := func(err error) {
		if err != nil {
			debug.PrintStack()
			t.Fatal()
		}
	}
	boolEqual := func(b bool) {
		if b != true {
			debug.PrintStack()
			t.Fatal()
		}
	}
	contain := func(a [][]byte, b []byte) {
		found := false
		for i := range a {
			if bytes.Equal(a[i], b) {
				found = true
				break
			}
		}
		if !found {
			debug.PrintStack()
			t.Fatal()
		}
	}
	c0, _, _, _ := NewConnection(2334, 10)
	c1, _, _, _ := NewConnection(1123, 10)
	c2, _, _, _ := NewConnection(1523, 10)
	control1 := make(chan bool)
	control2 := make(chan bool)
	boolEqual(c0.running)
	intEqual(len(c0.peers), 1)
	intEqual(0, c0.myId)
	go func() {
		<-control1
		myId, err := c1.Connect("localhost", 2334)
		nilEqual(err)
		intEqual(1, myId)
		control1 <- true

	}()
	go func() {
		<-control2
		myId, err := c2.Connect("localhost", 2334)
		nilEqual(err)
		intEqual(2, myId)
		control2 <- true

	}()
	control1 <- true
	boolEqual(<-control1)
	time.Sleep(time.Millisecond * 500)
	intEqual(c0.myId, 0)
	intEqual(c1.myId, 1)
	intEqual(len(c0.peers), 2)
	intEqual(len(c1.peers), 2)
	intEqual(len(c2.peers), 1)
	control2 <- true
	boolEqual(<-control2)
	time.Sleep(time.Millisecond * 500)
	intEqual(c0.myId, 0)
	intEqual(c1.myId, 1)
	intEqual(c2.myId, 2)
	intEqual(len(c0.peers), 3)
	intEqual(len(c1.peers), 3)
	intEqual(len(c2.peers), 3)
	c0.out <- []byte{0, 0, 1, 2, 3}
	c0.out <- []byte{1, 0, 1, 2, 3}
	c0.out <- []byte{2, 0, 1, 2, 3}
	c1.out <- []byte{0, 1, 2, 3, 4}
	c1.out <- []byte{1, 1, 2, 3, 4}
	c1.out <- []byte{2, 1, 2, 3, 4}
	c2.out <- []byte{0, 2, 3, 4, 5}
	c2.out <- []byte{1, 2, 3, 4, 5}
	c2.out <- []byte{2, 2, 3, 4, 5}
	c0expected := [][]byte{
		{0, 0, 1, 2, 3},
		{1, 1, 2, 3, 4},
		{2, 2, 3, 4, 5},
	}
	c1expected := [][]byte{
		{0, 0, 1, 2, 3},
		{1, 1, 2, 3, 4},
		{2, 2, 3, 4, 5},
	}
	c2expected := [][]byte{
		{0, 0, 1, 2, 3},
		{1, 1, 2, 3, 4},
		{2, 2, 3, 4, 5},
	}

	c0Rec := make([][]byte, 0)
	c1Rec := make([][]byte, 0)
	c2Rec := make([][]byte, 0)
	looping := true
	for looping {
		select {
		case msg := <-c0.in:
			c0Rec = append(c0Rec, msg)
		case msg := <-c1.in:
			c1Rec = append(c1Rec, msg)
		case msg := <-c2.in:
			c2Rec = append(c2Rec, msg)
		case <-time.After(time.Second):
			looping = false
		}
	}
	intEqual(len(c0Rec), 3)
	intEqual(len(c1Rec), 3)
	intEqual(len(c2Rec), 3)
	for i := range c0Rec {
		contain(c0expected, c0Rec[i])
	}
	for i := range c1Rec {
		contain(c1expected, c1Rec[i])
	}
	for i := range c2Rec {
		contain(c2expected, c2Rec[i])
	}

}
