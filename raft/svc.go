package raft

import (
	"net/rpc"
	"time"
)

func (rf *Raft) svcServer(idx int) {
	for {
		rf.mu.Lock()
		c, err := rpc.DialHTTP("tcp", rf.c.Servers[idx])
		if err != nil {
			rf.mu.Unlock()
			time.Sleep(1 * time.Second)
			continue
		}

		rf.peers[idx] = c
		rf.mu.Unlock()
	}
}
