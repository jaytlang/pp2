package kvraft

import (
	"math/rand"
	"sync"

	"pp2/raft"
)

type bcastID int

type bcast struct {
	mu   sync.Mutex
	id   int
	net  map[bcastID]chan raft.ApplyMsg
	nsub uint
}

func mkBcast(id int) *bcast {
	b := new(bcast)
	b.net = make(map[bcastID]chan raft.ApplyMsg)
	b.id = id
	return b
}

func (b *bcast) pub(kv *KVServer, evt raft.ApplyMsg, ack chan bool) {
	b.mu.Lock()

	//fmt.Printf("KV: BCAST(%d): Publishing operation %d to %d clients\n", b.id, evt.CommandIndex, len(b.net))
	for _, c := range b.net {
		if kv.killed() {
			break
		}
		//fmt.Printf("\t-> %d\n", id)
		c <- evt
	}

	// Ensure pub is not called again until the unsubscribe
	// operation completes
	cnl := len(b.net)
	b.mu.Unlock()
	for i := 0; i < cnl; i++ {
		<-ack
	}
	//fmt.Printf("KV: BCAST(%d): All publishes acked\n", b.id)
}

func (b *bcast) sub(c chan raft.ApplyMsg) bcastID {
	b.mu.Lock()
	defer b.mu.Unlock()
	var thisID bcastID

	for {
		thisID = bcastID(rand.Int())
		if _, p := b.net[thisID]; !p {
			break
		}
	}

	b.net[thisID] = c
	//fmt.Printf("KV: BCAST(%d): New subscriber (%d). %d subs total\n", b.id, thisID, len(b.net))
	b.nsub++
	return thisID
}

func (b *bcast) unsub(thisID bcastID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nsub--
	close(b.net[thisID])
	delete(b.net, thisID)
}
