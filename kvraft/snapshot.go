package kvraft

import (
	"bytes"
	"fmt"
	"log"

	"pp2/labgob"
	"pp2/raft"
)

type Snapshot struct {
	Kvm       map[string]string
	Unwritten map[uint]bool
	Unseen    map[uint]bool
	Last      map[uint]uint
}

// All snapshot functions expect locks held

func (kv *KVServer) shouldSnapshot() bool {
	if kv.maxraftsz < 0 {
		return false
	}
	rsz := kv.rf.Persister.RaftStateSize()
	return rsz >= kv.maxraftsz-1
}

// Raft calls rf.persist() so we don't need to worry
// about getting anything onto disk. We just need to make
// a snapshot, marshal it into bits and bytes, and go on
// with our day
func (kv *KVServer) takeSnapshot() {
	s := Snapshot{
		Kvm:       kv.kvm,
		Unwritten: kv.unwritten,
		Unseen:    kv.unseen,
		Last:      kv.last,
	}

	b := bytes.NewBuffer([]byte{})
	e := labgob.NewEncoder(b)
	if e.Encode(&s) != nil {
		log.Fatal("Get better encode error checking!")
	}

	// b now contains the snapshot
	// send it along to raft, which will always
	// succeed
	kv.rf.Snapshot(kv.topidx, b.Bytes())
	fmt.Printf("KV: SS: Snapshot taken to index %d\n", kv.topidx)
}

// Sometimes works, sometimes bails out
// Assumes SnapshotValid etc.
func (kv *KVServer) considerApplyMsg(v *raft.ApplyMsg) {
	// Pull the snapshot off the wire
	s := Snapshot{}
	b := bytes.NewBuffer(v.Snapshot)
	d := labgob.NewDecoder(b)
	if d.Decode(&s) != nil {
		log.Fatal("Get better decode error checking!")
	}

	if kv.rf.CondInstallSnapshot(v.SnapshotTerm, v.SnapshotIndex, v.Snapshot) {
		// We succeeded, update our own state
		// and flush out the last dict
		kv.last = s.Last
		kv.kvm = s.Kvm
		kv.unseen = s.Unseen
		kv.unwritten = s.Unwritten
		fmt.Printf("KV: SS: Snapshot conditionally installed to index %d\n", v.SnapshotIndex)
	} else {
		fmt.Printf("KV: SS: Snapshot failed to install to index %d\n", v.SnapshotIndex)
	}
}

// Called on startup
func (kv *KVServer) pullLatestSnapshot(p *raft.Persister) {
	ss := p.ReadRaftState()
	if ss == nil || len(ss) < 1 {
		return
	}

	// Skip ahead, skip ahead
	var t int
	var vf int
	var lg []*raft.LogEntry

	var si int
	var snapshot []byte

	buf := bytes.NewBuffer(ss)
	d := labgob.NewDecoder(buf)

	d.Decode(&t)
	d.Decode(&vf)
	err1 := d.Decode(&si)
	d.Decode(&lg)
	err2 := d.Decode(&snapshot)
	if err1 != nil || err2 != nil {
		log.Fatal("Get better decode error checking 1!")
	}

	s := Snapshot{}
	buf = bytes.NewBuffer(snapshot)
	d = labgob.NewDecoder(buf)
	if d.Decode(&s) != nil {
		log.Fatal("Get better decode error checking!")
	}

	kv.kvm = s.Kvm
	kv.unseen = s.Unseen
	kv.unwritten = s.Unwritten
	kv.last = s.Last
	fmt.Printf("KV: SS: Pulled snapshot during restart\n")
}
