package src

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Connection interface {
	Connect(ip string, port int) (int, error)
	Close()
}

var _ Connection = new(connection)

type connection struct {
	id     int
	port   int
	running  bool
	group    *sync.WaitGroup
	listener *net.TCPListener
	peers    []*peer
	in, out  chan []byte
}

type peer struct {
	id   int
	ip   string
	port int
	conn *net.TCPConn
}

//setup a new connection
func NewConnection(port int, bufferSize int) (*connection, <-chan []byte, chan<- []byte, error) {
	conn := new(connection)
	conn.in, conn.out = make(chan []byte, 1000), make(chan []byte, 1000)
	conn.port = port
	conn.peers = make([]*peer, 1)
	conn.running = true
	conn.group = new(sync.WaitGroup)
	
	listener, err := net.Listen("tcp", fmt.Sprint(":", port))
	if err != nil {
		return nil, nil, nil, err
	}
	conn.listener = listener.(*net.TCPListener)
	go conn.listen()
	go conn.sendLoop()
	return conn, conn.in, conn.out, nil
}


//connect host to another host
func (c *connection) Connect(ip string, port int) (int, error) {
	var err error
	var tempConn net.Conn
	tempConn, err = net.DialTimeout("tcp", fmt.Sprint(ip, ":", port), time.Second*5)
	if err != nil {
		panic("Error: Dial TCP error")
	}
	tmpConn := tempConn.(*net.TCPConn)
	if err != nil {
		panic("Couldnt dial the host: " + err.Error())
	}

	write(tmpConn, []byte{0, 0, byte(c.port / 256), byte(c.port % 256)})
	
	msg, _ := read(tmpConn)
	c.id = int(msg[0])
	c.addPeer(c.id, nil, 0)
	c.peers = make([]*peer, c.id+1)
	otherId := int(msg[1])
	c.addPeer(otherId, tmpConn, port)
	j := 2
	for j < len(msg) {
		id := int(msg[j])
		j++
		ip, port, k := addrFromBytes(msg[j:])
		newConn, err := net.Dial("tcp", fmt.Sprint(ip, ":", port))
		if err != nil {
			panic("Error: Connect to address " + fmt.Sprint(ip, ":", port) + ": " + err.Error())
		}
		write(newConn, []byte{0, byte(c.id), byte(c.port / 256), byte(c.port % 256)})
		c.addPeer(id, newConn.(*net.TCPConn), port)
		go c.receive(c.peers[id])
		j += k
	}
	go c.receive(c.peers[otherId])
	return c.id, nil
}

func (c *connection) Close() {
	close(c.out)
	c.group.Wait()
}

// keep sending message
func (c *connection) sendLoop() {
	c.group.Add(1)
	var id int
	for msg := range c.out {
		id = int(msg[0])
		if id != c.id {
			if id < len(c.peers) {
				msg[0] = 1

				write(c.peers[id].conn, msg)
			} else {
				go func() {
					time.Sleep(time.Millisecond * 500)
					c.out <- msg
				}()
			} 
		} else {
			c.in <- msg
		}
	}
	c.running = false
	c.group.Done()
	c.group.Wait()
	close(c.in)
}


func (c *connection) receive(peer *peer) {
	c.group.Add(1)
	for c.running {

		msg, _ := read(peer.conn)
		if msg == nil {
			continue
		}

		if len(msg) > 0 && msg[0] == 0 {
			id := int(msg[1])
			ip, port, _ := addrFromBytes(msg[2:])
			conn := c.connectToHost(ip, port)
			c.addPeer(id, conn, port)
		} else {
			newMsg := append([]byte{byte(peer.id)}, msg[1:]...)
			c.in <- newMsg

		}
	}
	peer.conn.Close()
	c.group.Done()
}

func (c *connection) listen() {
	c.group.Add(1)

	for c.running {
		c.listener.SetDeadline(time.Now().Add(time.Millisecond * 500))
		conn, err := c.listener.AcceptTCP()
		if err == nil {
			c.addHost(conn)
		} 
	}
	c.listener.Close()
	c.group.Done()
}


func (c *connection) addHost(conn *net.TCPConn) {
	msg, _ := read(conn)
	var buf bytes.Buffer
	id := int(msg[1])
	port := int(msg[2])*256 + int(msg[3])
	if id == 0 {
		id = len(c.peers)
		buf.Write([]byte{byte(id), byte(c.id)})
		for i := range c.peers {
			if i != c.id {
				buf.WriteByte(byte(i))
				buf.Write(addrToBytes(c.peers[i].ip, c.peers[i].port))
			}
		}
		write(conn, buf.Bytes())
	}
	c.addPeer(id, conn, port)
	go c.receive(c.peers[id])
}


func (c *connection) connectToHost(ip string, port int) *net.TCPConn {
	var conn net.Conn
	var err error
	for c.running {
		conn, err = net.DialTimeout("tcp", fmt.Sprint(ip, ":", port), time.Millisecond*500)
		if err != nil {
			panic("Error: connect to host fail")
		}
	}
	return conn.(*net.TCPConn)
}

func (c *connection) getAddr(id int) string {
	peer := c.peers[id]
	return fmt.Sprint(peer.ip, ":", peer.port)
}

func (c *connection) addPeer(id int, conn *net.TCPConn, port int) {
	if len(c.peers) <= id {
		c.peers = append(c.peers, make([]*peer, (id-len(c.peers))+1)...)
	}
	ip := "localhost"
	if conn != nil {
		addr := conn.RemoteAddr().String()
		splitAddr := strings.Split(addr, ":")
		if len(splitAddr) == 2 {
			ip = splitAddr[0]
		}
	}

	c.peers[id] = &peer{id, ip, port, conn}
}

func read(conn net.Conn) ([]byte, error) {
	length := make([]byte, 8)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, err := io.ReadFull(conn, length)
	if err != nil {
		return nil, err
	}
	l, _ := binary.Varint(length)
	msg := make([]byte, l)
	_, err = io.ReadFull(conn, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func write(conn net.Conn, data []byte) {
	length := uint64(len(data))

	l := make([]byte, 8)
	binary.PutVarint(l, int64(length))
	msg := append(l, data...)
	n := 0
	var err error
	for {
		n, err = conn.Write(msg[n:])
		if err == nil {
			break
		}
		panic(err.Error())
	}
}



func addrFromBytes(b []byte) (string, int, int) {
	s := make([]string, 4)
	for i := 0; i < 4; i++ {
		s[i] = strconv.Itoa(int(b[i]))
	}
	ip := strings.Join(s, ".")
	port, i := binary.Varint(b[4:])
	return ip, int(port), i + 4
}

func addrToBytes(ip string, port int) []byte {
	ipArray := strings.Split(ip, ".")
	//localhost
	if len(ipArray) != 4 {
		ipArray = []string{"127", "0", "0", "1"}
	}
	ipInBytes := make([]byte, 4)
	for i := range ipArray {
		v, _ := strconv.Atoi(ipArray[i])
		ipInBytes[i] = byte(v)
	}
	buf := make([]byte, binary.Size(int64(port)))
	i := binary.PutVarint(buf, int64(port))
	result := append(ipInBytes, buf[:i]...)
	return result
}
