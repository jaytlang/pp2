package kvraft

import (
	"fmt"
	"math/rand"
	"net/rpc"
	"pp2/netdrv"
	"sync"
	"time"
)

type Clerk struct {
	mu      sync.Mutex
	id      uint
	servers []*rpc.Client
	lastLdr int
	c       *netdrv.NetConfig
}

// Utility functions...
func MakeClerk(c *netdrv.NetConfig) *Clerk {
	ck := new(Clerk)
	ck.servers = c.DialAll()
	ck.c = c
	ck.id = ck.mkSeq()
	return ck
}

func (ck *Clerk) mkSeq() uint {
	return uint(rand.Uint64())
}

func (ck *Clerk) doRequest(op OpCode, key string, value string) string {
	a := new(RequestArgs)

	ck.mu.Lock()
	defer ck.mu.Unlock()

	s := ck.mkSeq()
	l := ck.lastLdr
	npeers := len(ck.servers)

	ck.mu.Unlock()
	a.ClientId = ck.id
	a.Seq = s
	a.Code = op
	a.Key = key
	a.Value = value

	fmt.Printf("KV: C: Submitting request %v for %s/%s\n", op, key, value)

	for {
		r := new(RequestReply)
		rc := make(chan error)
		go func() {
			// Fails immediately if server is down
			rc <- ck.servers[l].Call("KVServer.Request", a, r)
		}()

		select {
		case ok := <-rc:
			if ok != nil {
				ck.mu.Lock()
				// Spammy
				c, err := rpc.DialHTTP("tcp", ck.c.Servers[l])
				if err != nil {
					ck.servers[l] = c
				}
				ck.mu.Unlock()
			}
			if ok != nil || r.E == ErrWrongLeader || r.E == ErrTimeout {
				goto retry
			} else {
				fmt.Printf("KV: C: Request %v -> %s/%s finished successfully\n", op, key, value)
				ck.mu.Lock()
				ck.lastLdr = l
				return r.Value
			}

		case <-time.After(time.Second):
			goto retry
		}
	retry:
		fmt.Printf("KV: C: Retrying operation %s/%s\n", key, value)
		l++
		if l >= npeers {
			l = 0
		}

	}
}

func (ck *Clerk) Get(key string) string {
	return ck.doRequest(GetOp, key, "")
}
func (ck *Clerk) Put(key string, value string) {
	ck.doRequest(PutOp, key, value)
}
func (ck *Clerk) Append(key string, value string) {
	ck.doRequest(AppendOp, key, value)
}
