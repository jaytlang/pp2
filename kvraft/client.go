package kvraft

import (
	"errors"
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
	rand.Seed(time.Now().Unix())
	ck := new(Clerk)
	ck.servers = c.DialAll()
	ck.c = c
	ck.id = ck.mkSeq()
	return ck
}

func (ck *Clerk) mkSeq() uint {
	return uint(rand.Uint64())
}

func (ck *Clerk) doRequest(op OpCode, key string, value string) (string, error) {
	a := new(RequestArgs)

	ck.mu.Lock()

	s := ck.mkSeq()
	l := ck.lastLdr
	npeers := len(ck.servers)

	ck.mu.Unlock()
	a.ClientId = ck.id
	a.Seq = s
	a.Code = op
	a.Key = key
	a.Value = value

	//fmt.Printf("KV: C: Submitting request %v for %s/%s\n", op, key, value)

	for {
		r := new(RequestReply)
		rc := make(chan error)
		go func(idx int) {
			// Fails immediately if server is down
			ck.mu.Lock()
			res := ck.servers[idx].Call("KVServer.Request", a, r)
			ck.mu.Unlock()
			rc <- res
		}(l)

		select {
		case ok := <-rc:
			if ok != nil {
				ck.mu.Lock()
				c, err := rpc.DialHTTP("tcp", ck.c.Servers[l])
				if err == nil {
					ck.servers[l] = c
				}
				ck.mu.Unlock()
			}
			if ok != nil || r.E == ErrWrongLeader || r.E == ErrTimeout {
				goto retry
			} else if r.E == ErrLockHeld {
				// Reset sequence number and try again
				a.Seq = ck.mkSeq()
				time.Sleep(500 * time.Millisecond)
				a.Value = fmt.Sprintf("%d", time.Now().Unix())
				goto retry
			} else if r.E == ErrLockNotHeld {
				//fmt.Printf("KV: C: Failed to execute operation %v, lock not held\n", op)
				return "", errors.New("lock not held")
			} else {
				//fmt.Printf("KV: C: Request %v -> %s/%s finished successfully\n", op, key, value)
				ck.mu.Lock()
				ck.lastLdr = l
				ck.mu.Unlock()
				return r.Value, nil
			}

		case <-time.After(time.Second):
			goto retry
		}
	retry:
		//fmt.Printf("KV: C: Retrying operation %s/%s\n", key, value)
		l++
		if l >= npeers {
			l = 0
		}

		time.Sleep(1 * time.Millisecond)
	}
}

func (ck *Clerk) Get(key string) (string, error) {
	return ck.doRequest(GetOp, key, "")
}
func (ck *Clerk) Put(key string, value string) error {
	_, err := ck.doRequest(PutOp, key, value)
	return err
}
func (ck *Clerk) Append(key string, value string) error {
	_, err := ck.doRequest(AppendOp, key, value)
	return err
}

func (ck *Clerk) Acquire(lockk string) {
	ck.doRequest(AcquireOp, lockk, fmt.Sprintf("%d", time.Now().Unix()))
}

func (ck *Clerk) Release(lockk string) error {
	_, err := ck.doRequest(ReleaseOp, lockk, "")
	return err
}

func (ck *Clerk) Renew(lockk string) error {
	_, err := ck.doRequest(RenewOp, lockk, fmt.Sprintf("%d", time.Now().Unix()))
	return err
}

func (ck *Clerk) WriteToFile(key string, value string) error {
	_, err := ck.doRequest(WriteToFileOp, key, value)
  return err
}

func (ck *Clerk) ReadFromFile(key string) (string, error) {
	return ck.doRequest(ReadFromFileOp, key, "")
}
