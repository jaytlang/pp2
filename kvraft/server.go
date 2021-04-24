package kvraft

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"pp2/labgob"
	"pp2/netdrv"
	"pp2/raft"
)

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	ackCh   chan bool
	dead    int32 // set by Kill()

	// Lab 3A: sequence numbers and the map
	kvm map[string]string

	unwritten map[uint]bool
	unseen    map[uint]bool
	last      map[uint]uint
	b         *bcast

	// Lab 3B: snapshotting infos and more
	maxraftsz int
	topidx    int
}

func (kv *KVServer) checkHoldLock(cmd RequestArgs) bool {
  ov := kv.kvm["lock_"+cmd.Key]
  if ov == "" {
    return false
  }

  cid, _ := strconv.Atoi(strings.Split(ov, "/")[0])
  return cid == int(cmd.ClientId)
}

// Helpers for the actual map
// Responsible for reply.Value, reply.E
func (kv *KVServer) commitOp(cmd RequestArgs) error {
	switch cmd.Code {
  // XXX: error checking for not holding lock?
	case PutOp:
    if kv.checkHoldLock(cmd) {
      kv.kvm[cmd.Key] = cmd.Value
    }
	case AppendOp:
    if kv.checkHoldLock(cmd) {
      kv.kvm[cmd.Key] += cmd.Value
    }
	case AcquireOp:
		ov := kv.kvm["lock_"+cmd.Key]
		if ov != "" {
			ts, _ := strconv.Atoi(strings.Split(ov, "/")[1])
			if nt, _ := strconv.Atoi(cmd.Value); nt-ts > lockLeaseTime {
				goto doAcquire
			}
			return errors.New("failed to acquire")
		}

		// Acquire
	doAcquire:
		kv.kvm["lock_"+cmd.Key] = fmt.Sprintf("%d/%s", cmd.ClientId, cmd.Value)

	case ReleaseOp:
		kv.kvm["lock_"+cmd.Key] = ""
	}
	return nil
}

func (kv *KVServer) Request(args *RequestArgs, reply *RequestReply) error {
	// Set up the client sequence number
	// off the bat, in case things go awry
	reply.Seq = args.Seq

	// Subscribe
	msgc := make(chan raft.ApplyMsg)

rerequest:
	// Push the request through to our server
	idx, _, isLdr := kv.rf.Start(args)
	if !isLdr {
		reply.E = ErrWrongLeader
		return nil
	}
	id := kv.b.sub(msgc)
	fmt.Printf("KV: %d: Request %s/%s assigned this ID\n", id, args.Key, args.Value)
	for !kv.killed() {
		v := <-msgc
		// Sanity check: are we the leader?
		if _, amLeader := kv.rf.GetState(); !amLeader {
			kv.b.unsub(id)
			kv.ackCh <- true
			reply.E = ErrWrongLeader
			return nil
		}

		// Check if the message goes where ours
		// should go, using the client's sequence as a unique
		// identifier (without temporal significance)
		if v.CommandIndex == idx {
			cmd := v.Command.(*RequestArgs)
			if cmd.Seq != args.Seq {
				fmt.Printf("KV: %d: Re-requesting operation\n", id)
				kv.b.unsub(id)
				kv.ackCh <- true
				goto rerequest
			}
			kv.b.unsub(id)
			kv.ackCh <- true

			// If the command was a get command, give them
			// either the most up to date key or ErrNoKey
			kv.mu.Lock()
			defer kv.mu.Unlock()

			if args.Code == GetOp {
				if k, ok := kv.kvm[args.Key]; ok {
					reply.Value = k
					reply.E = OK
				} else {
					reply.E = ErrNoKey
				}
			} else if cmd.Code == FailingAcquireOp {
				reply.E = ErrLockHeld
			} else {
				reply.E = OK
			}
			fmt.Printf("KV: %d: Requested operation %s/%s completed\n", id, args.Key, args.Value)
			return nil
		} else {
			fmt.Printf("KV: %d: Wrong index (expected %d, got %d)\n", id, idx, v.CommandIndex)
			kv.ackCh <- true
		}
	}

	return nil
}

//
// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

func (kv *KVServer) manageApplyCh() {
	for !kv.killed() {
		v := <-kv.applyCh

		// Ensure the commit occurs before others
		// know about it
		if v.SnapshotValid {
			fmt.Printf("KV: SS: Valid snapshot detected\n")
			kv.mu.Lock()
			kv.considerApplyMsg(&v)
			kv.mu.Unlock()

		} else if v.CommandValid {
			cmd := *v.Command.(*RequestArgs)
			fmt.Printf("KV: MCH: Got operation %d...\n", v.CommandIndex)
			kv.mu.Lock()

			// If the client sent something before this...
			if v, ok := kv.last[cmd.ClientId]; ok {
				// If that something is not this command, implying
				// they received the previous command...
				if v != cmd.Seq {
					// They have seen the last command
					delete(kv.unseen, v)

					// The new "last" command is now this one
					kv.last[cmd.ClientId] = cmd.Seq
				}
			} else {
				// The client did not send something before
				// this, so their last command is this
				kv.last[cmd.ClientId] = cmd.Seq
			}

			// If this something doesn't appear in our records,
			// add it to our records so that a client can ack it later
			// Partial overlap with the above but works as tested
			if _, ok := kv.unwritten[cmd.Seq]; !ok {
				if _, ok := kv.unseen[cmd.Seq]; !ok {
					kv.unwritten[cmd.Seq] = true
					kv.unseen[cmd.Seq] = true
				}
			}

			fmt.Printf("KV: DBG: Size of unseen: %d\n", len(kv.unseen))

			// Has the op been written?
			if _, ok := kv.unwritten[cmd.Seq]; ok {
				fmt.Printf("KV: MCH: Writing operation %d\n", v.CommandIndex)
				err := kv.commitOp(cmd)
				if err != nil {
					if cmd.Code == AcquireOp {
						fmt.Printf("KV: LOCK: Note that acquire failed!\n")
						cmd.Code = FailingAcquireOp
						v.Command = &cmd
					}
				}

				delete(kv.unwritten, cmd.Seq)
			} else {
				// if the op has been written, and it's an Acquire/Release,
				// we switch the opcode based on whether the person in question
				// holds the lock at the time of checking. this ensures consistency
				// from their end.
				if cmd.Code == AcquireOp {
					cc := strings.Split(kv.kvm["lock_"+cmd.Key], "/")[0]
					if cc != fmt.Sprintf("%d", cmd.ClientId) {
						cmd.Code = FailingAcquireOp
						v.Command = &cmd
					}
				}
			}

			// Before we go, snapshots!
			kv.topidx = v.CommandIndex
			if kv.shouldSnapshot() {
				kv.takeSnapshot()
			}

			// Has the op been published?
			if _, ok := kv.unseen[cmd.Seq]; ok {
				kv.mu.Unlock()
				kv.b.pub(kv, v, kv.ackCh)
			} else {
				kv.mu.Unlock()
			}
		}
	}
}

func StartKVServer(c *netdrv.NetConfig, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(&RequestArgs{})
	labgob.Register(&Snapshot{})

	kv := new(KVServer)
	kv.me = me
	kv.applyCh = make(chan raft.ApplyMsg)
	kv.ackCh = make(chan bool)

	kv.kvm = make(map[string]string)
	kv.unseen = make(map[uint]bool)
	kv.unwritten = make(map[uint]bool)
	kv.last = make(map[uint]uint)

	kv.pullLatestSnapshot(persister)
	kv.rf = raft.Make(c, me, persister, kv.applyCh)

	kv.b = mkBcast(kv.me)
	kv.maxraftsz = maxraftstate

	// Ready to rock. Set up RPC.
	s := rpc.NewServer()
	s.Register(kv)
	s.HandleHTTP("/kvr", "/kvdb")

	// KV serves on 1235, raft on 1234
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", c.KVPort))
	if e != nil {
		log.Fatal("listen error:", e)
	}

	go http.Serve(l, s)
	fmt.Printf("KV: DBG: SERVING HTTP NOW OVER 1235\n")
	go kv.manageApplyCh()

	return kv
}
