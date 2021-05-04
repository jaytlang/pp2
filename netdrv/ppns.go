package netdrv

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"time"
)

type NameServer struct {
	lock  sync.Mutex
	names map[int]string

	maxName int
}

type NsRqArgs struct {
	Register bool
	Name     int
	Address  string
}

type NsRpArgs struct {
	OK      bool
	Name    int
	Address string
}

const nsPort = 1233

func (n *NameServer) mkName() int {
	n.maxName++
	return n.maxName - 1
}

func (n *NameServer) Request(args *NsRqArgs, reply *NsRpArgs) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if args.Register {
		nn := n.mkName()
		n.names[nn] = args.Address
		reply.OK = true
		reply.Name = nn
		return nil

	} else if a, ok := n.names[args.Name]; ok {
		reply.Address = a
		reply.OK = true
		return nil

	} else {
		reply.OK = false
		return nil
	}
}

func RunNameserver() {
	n := new(NameServer)
	n.names = make(map[int]string)

	s := rpc.NewServer()
	s.Register(n)
	s.HandleHTTP("/ns", "/nsdb")

	// KV serves on 1235, raft on 1234, ns on 1233
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", nsPort))
	if e != nil {
		log.Fatal("listen error:", e)
	}

	fmt.Printf("Running nameserver on %s\n", getMyIp())
	http.Serve(l, s)
}

func getName(nsAddr string, which int) string {
	c, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", nsAddr, nsPort))
	if err != nil {
		log.Fatal("couldn't dial nameserver:", err)
	}

retry:
	args := &NsRqArgs{
		Register: false,
		Name:     which,
	}
	reply := &NsRpArgs{}

	if c.Call("NameServer.Request", args, reply) != nil {
		log.Fatal("couldn't get name:", err)
	} else if !reply.OK {
		time.Sleep(1 * time.Second)
		goto retry
	}

	return reply.Address
}

// Hacks
func getMyIp() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func registerName(nsAddr string) int {
	c, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", nsAddr, nsPort))
	if err != nil {
		log.Fatal("couldn't dial nameserver:", err)
	}

	args := &NsRqArgs{
		Register: true,
		Address:  getMyIp(),
	}
	reply := &NsRpArgs{}
	if c.Call("NameServer.Request", args, reply) != nil {
		log.Fatal("couldn't get name:", err)
	}
	return reply.Name
}
