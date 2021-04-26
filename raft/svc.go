package raft

import (
	"net/rpc"
	"time"
)

func (rf *Raft) svcServer(idx int) {
	for {
		rf.mu.Lock()
		//fmt.Printf("%d: %d: Attempting to redial server %d\n", rf.me, rf.term, idx)
		c, err := rpc.DialHTTP("tcp", rf.c.Servers[idx])
		if err != nil {
			//fmt.Printf("%d: %d: Failed to hit server %d\n", rf.me, rf.term, idx)
			rf.mu.Unlock()
			time.Sleep(1 * time.Second)
			continue
		}

		//fmt.Printf("%d: %d: Redialed server %d\n", rf.me, rf.term, idx)
		rf.peers[idx] = c
		rf.mu.Unlock()
		break
	}
}
