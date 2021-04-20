package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"sync/atomic"
	"time"

	"pp2/labgob"
	"pp2/netdrv"
)

// States a raft server can be in
type State int

// LogEntry type is sent over the
// wire, so all fields must be public
type LogEntry struct {
	Term int
	Cmd  interface{}
}

const (
	leader    State = iota
	follower  State = iota
	candidate State = iota
)

// Le object itself
type Raft struct {
	mu        sync.Mutex    // Lock to protect shared access to this peer's state
	peers     []*rpc.Client // RPC end points of all peers
	Persister *Persister    // Object to hold this peer's persisted state
	me        int           // this peer's index into peers[]
	dead      int32         // set by Kill()

	term     int
	votedFor int
	st       State
	ts       int64

	// Lab 2B: all servers
	log        []*LogEntry
	commitIdx  int
	appliedIdx int
	lc         chan ApplyMsg
	disam      chan ApplyMsg

	// Lab 2B: leader only, re-init
	nextForServer []int
	atServer      []int
	msgFlag       bool

	// Lab 2D: snapshotting
	snapshot      []byte
	snapshotIndex int
}

type voteCounter struct {
	mu     sync.Mutex
	votes  int
	killed bool
}

// Intervals -- should produce sane results
// One heartbeat per roughly 1/9 second, and
// wait anywhere between 1x and 3x that before
// sounding an alarm re. a missed heartbeat
const hbMs = 101
const minWait = hbMs * 2
const maxWait = hbMs * 4

// Default lab functions and lab-accessed
// calls and top level RPC handlers below
// this point

func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// term, isLeader
	return rf.term, (rf.st == leader)
}

type RequestVoteArgs struct {
	// Lab 2A
	Term        int
	CandidateId int

	// Lab 2B
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	// Lab 2A
	Term     int
	LeaderId int

	// Lab 2B
	LastLogIndex    int
	LastLogTerm     int
	Entries         []LogEntry
	LeaderCommitIdx int
}

type AppendEntriesReply struct {
	// Lab 2A/2B
	Term    int
	Success bool

	// Lab 2C
	XTerm  int
	XIndex int
	XLen   int
}

type InstallSnapshotArgs struct {
	Term     int
	LeaderId int

	LastIncludedIndex int
	LastIncludedTerm  int
	Data              []byte
}

type InstallSnapshotReply struct {
	Term int
}

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// msgCommon: become a follower and fall in line
	// with the current term if need be.
	res := rf.msgCommon(args.Term)

	if !res {
		fmt.Printf("%d: %d: RV new term, resetting my vote\n", rf.me, rf.term)
		rf.votedFor = -1
		rf.persist()
	}

	reply.Term = rf.term
	if args.Term < rf.term {
		fmt.Printf("%d: %d: Received bad term for requestVote RPC\n", rf.me, rf.term)
		reply.VoteGranted = false
		return nil
	}

	// Next, let's check the log
	if !rf.otherIsRecent(args.LastLogIndex, args.LastLogTerm) {
		fmt.Printf("%d: %d: Candidate log is not up to date\n", rf.me, rf.term)
		reply.VoteGranted = false
		return nil
	}

	// The terms align. Now let's see if we've
	// already voted for somebody
	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		fmt.Printf("%d: %d: Received requestVote RPC and granted vote to %d\n", rf.me, rf.term, args.CandidateId)
		rf.ts = mkTimeMs()

		rf.votedFor = args.CandidateId
		rf.persist()
		reply.VoteGranted = true
	} else {
		fmt.Printf("%d: %d: Declined requestVote RPC for %d\n", rf.me, rf.term, args.CandidateId)
		reply.VoteGranted = false
	}
	return nil
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) error {
	// msgCommon: we DEFINITELY need to make sure we're
	// a follower in this case. This will cause us to fall
	// in line if we for some reason aren't already...it's also
	// possible we have a masquerading leader
	rf.mu.Lock()
	defer rf.mu.Unlock()

	res := rf.msgCommon(args.Term)

	// Term check
	if !res {
		fmt.Printf("%d: %d: New term and am follower\n", rf.me, rf.term)
	}

	reply.Term = rf.term
	reply.Success = true

	reply.XLen = -1
	reply.XTerm = -1
	reply.XIndex = -1

	// Will trigger msgcommon
	if args.Term < rf.term {
		fmt.Printf("%d: %d: Got bad heartbeat term\n", rf.me, rf.term)
		reply.XLen = -1
		goto fail
	}

	rf.ts = mkTimeMs()
	rf.st = follower

	// Do we have a log entry at LastLogIndex?
	// If no, this is XLen case. No change necessary
	if len(rf.log) <= args.LastLogIndex {
		reply.XLen = len(rf.log)
		fmt.Printf("%d: %d: No entry at LastLogIndex=%d\n", rf.me, rf.term, args.LastLogIndex)
		goto fail
	}

	// Does that log entry have the right term?
	// This would constitute a collision, so we should update
	// XTerm/XIndex
	fmt.Printf("%d: %d: Checking LastLogIndex %d\n", rf.me, rf.term, args.LastLogIndex)
	if rf.log[args.LastLogIndex] == nil {
		goto fail
	}
	if rf.log[args.LastLogIndex].Term != args.LastLogTerm {
		fmt.Printf("%d: %d: Wrong term at flwr LastLogIndex\n", rf.me, rf.term)
		reply.XTerm = rf.log[args.LastLogIndex].Term
		reply.XIndex = rf.firstIndexForTerm(reply.XTerm)
		goto fail
	}

	// Do we have a log entry at LastLogIndex + 1?
	// If so, keep going but delete trailing items
	if len(rf.log) > args.LastLogIndex+1 {
		fmt.Printf("%d: %d: Overwriting old log entries\n", rf.me, rf.term)
		rf.log = rf.log[:args.LastLogIndex+1]
		rf.persist()
	}

	// Append new log entries
	if len(args.Entries) > 0 {
		fmt.Printf("%d: %d: Got AppendEntries RPC\n", rf.me, rf.term)
		for i := range args.Entries {
			newEnt := args.Entries[i]
			rf.log = append(rf.log, &newEnt)
		}
		rf.persist()
	} else {
		fmt.Printf("%d: %d: Got heartbeat RPC\n", rf.me, rf.term)
	}

	// Increment commit index if needed
	if args.LeaderCommitIdx > rf.commitIdx {
		oldCommitIdx := rf.commitIdx
		lastEnt := len(rf.log) - 1
		if lastEnt < args.LeaderCommitIdx {
			rf.commitIdx = lastEnt
		} else {
			rf.commitIdx = args.LeaderCommitIdx
		}
		fmt.Printf("%d: %d: Flwr committed new entries to index %d from %d\n", rf.me, rf.term, rf.commitIdx, oldCommitIdx)
		rf.applyOutstanding()
	}

	return nil
fail:
	reply.Success = false
	return nil
}

func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs, reply *InstallSnapshotReply) error {
	rf.mu.Lock()

	// If msgTerm is GREATER than our term, fall in line
	res := rf.msgCommon(args.Term)
	if !res {
		fmt.Printf("%d: %d: SS: [NEW] IS new term, fell into line\n", rf.me, rf.term)
	}

	// If msgTerm is LESS than our term, reply immediately to update the "leader"
	reply.Term = rf.term
	if args.Term < rf.term {
		fmt.Printf("%d: %d: SS: [NEW] Received bad term for IS RPC\n", rf.me, rf.term)
		rf.mu.Unlock()
		return nil
	}

	// Otherwise, hand the service the snapshot. It will come back to us
	// and have us install it, which comprises the back half of this RPC.
	// No need to duplicate logic

	rf.ts = mkTimeMs()
	fmt.Printf("%d: %d: SS: [NEW] Got IS RPC, installing...\n", rf.me, rf.term)
	rf.mu.Unlock()
	a := ApplyMsg{
		SnapshotValid: true,
		Snapshot:      args.Data,
		SnapshotIndex: args.LastIncludedIndex,
		SnapshotTerm:  args.LastIncludedTerm,
	}

	// Bye!
	rf.disam <- a
	return nil
}

// For these two, we hold the lock already
// Alternatively, operating under the assumption
// that rf.peers is NEVER changed, we don't need
// to hold the lock here, which allows parallelization
func (rf *Raft) doRV(server int, args *RequestVoteArgs, reply *RequestVoteReply) error {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) doAE(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) error {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) doIS(server int, args *InstallSnapshotArgs, reply *InstallSnapshotReply) error {
	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)
	return ok
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. use killed() to check if we're dead.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) ticker() {

	for !rf.killed() {
		// First run: depending on whether we are
		// a follower or a leader, do the sleep and
		// update the current time
		rf.mu.Lock()
		rf.ts = mkTimeMs()
		currentTime := rf.ts

		if rf.st == follower {
			rf.mu.Unlock()
			sleepUntilTimeout()
		} else {
			sleepDuration := hbMs * time.Millisecond
			if rf.msgFlag {
				sleepDuration /= 4
				rf.msgFlag = false
			}
			rf.mu.Unlock()

			// If we have word from Start that something is
			// happening, sleep for 1/5 this period to allow
			// other requests to complete
			time.Sleep(sleepDuration)
		}

		// Sleep is done. Reacquire mutex for stage 2...
		rf.mu.Lock()

		// If we are a follower, check to ensure that
		// the rf.ts has changed over the course of our nap
		// If it hasn't, declare candidacy up until we are
		// no longer a candidate. If we become the leader,
		// send a heartbeat
		if rf.st == follower {
			if rf.ts == currentTime {
				rf.st = candidate
				for rf.st == candidate && !rf.killed() {
					rf.mu.Unlock()
					rf.beCandidate()
					rf.mu.Lock()
				}
			}

			// If we are the leader, just get it over with
			// and send a heartbeat. No additional work required
			// Will fall through from above if we're elected
		}
		if rf.st == leader {
			if rf.ts == currentTime {
				rf.doHeartbeat()
			}
		}

		rf.mu.Unlock()
	}

}

func Make(c *netdrv.NetConfig, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.Persister = persister
	rf.me = me
	fmt.Printf("%d: X: Raft server is coming online\n", rf.me)

	// 2A initialization
	rf.term = 0
	rf.votedFor = -1
	rf.st = follower

	// 2B initialization
	rf.log = make([]*LogEntry, 1)
	rf.log[0] = &LogEntry{}
	rf.commitIdx = 0
	rf.appliedIdx = 0
	rf.lc = applyCh
	rf.disam = make(chan ApplyMsg, (1<<16)-1)

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// seed the RNG
	rand.Seed(time.Now().UnixNano())

	// Ready to rock. Set up RPC.
	rpc.Register(rf)
	rpc.HandleHTTP()

	// KV serves on 1235, raft on 1234
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", c.RaftPort))
	if err != nil {
		log.Fatal("listen error:", err)
	}

	// start ticker goroutine to start elections and watch commits
	go http.Serve(ln, nil)
	rf.peers = c.DialAll()

	fmt.Printf("%d: %d: Raft server is up and running\n", rf.me, rf.term)
	go rf.disambiguator()
	go rf.ticker()

	return rf
}

func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.st != leader {
		return -1, -1, false
	}

	n := LogEntry{Term: rf.term, Cmd: command}
	rf.log = append(rf.log, &n)
	rf.persist()
	fmt.Printf("%d: %d: Writing new log entry for idx %v\n", rf.me, rf.term, len(rf.log)-1)

	expectIdx := len(rf.log) - 1

	// Go go go! DELETE THIS
	rf.msgFlag = true
	return expectIdx, rf.term, true
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// called with lock held
func (rf *Raft) persist() {
	buf := bytes.Buffer{}
	enc := labgob.NewEncoder(&buf)

	err1 := enc.Encode(rf.term)
	err2 := enc.Encode(rf.votedFor)
	err3 := enc.Encode(rf.snapshotIndex)
	err4 := enc.Encode(rf.log[rf.snapshotIndex:])
	err5 := enc.Encode(rf.snapshot)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
		log.Fatal("Couldn't encode, get better error checking")
	}

	rf.Persister.SaveRaftState(buf.Bytes())
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 {
		return
	}

	buf := bytes.Buffer{}
	buf.Write(data)
	dec := labgob.NewDecoder(&buf)

	err1 := dec.Decode(&rf.term)
	err2 := dec.Decode(&rf.votedFor)
	err3 := dec.Decode(&rf.snapshotIndex)
	if err1 != nil || err2 != nil || err3 != nil {
		log.Fatal("Couldn't decode, get better error checking")
	}

	oldLog := make([]*LogEntry, rf.snapshotIndex)
	newLog := make([]*LogEntry, 0)

	err4 := dec.Decode(&newLog)
	err5 := dec.Decode(&rf.snapshot)
	if err4 != nil || err5 != nil {
		log.Fatal("Couldn't decode, get better error checking")
	}
	rf.log = append(oldLog, newLog...)
	rf.commitIdx = rf.snapshotIndex
	rf.appliedIdx = rf.snapshotIndex

	if rf.snapshotIndex > 0 {
		if rf.log[rf.snapshotIndex-1] != nil {
			log.Fatal("Erroneous decode")
		}
	}
	if rf.log[rf.snapshotIndex] == nil {
		log.Fatal("Erroneous decode")
	}

	fmt.Printf("%d: %d: [NEW] Read state. Log has valid entries from %d-%d\n", rf.me, rf.term, rf.snapshotIndex, len(rf.log)-1)
}

// A service wants to switch to snapshot.  Only do so if Raft hasn't
// had more recent info since it communicated the snapshot on applyCh.
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// Verify we aren't installing anything stale. If the snapshot is
	// not at least as recent as we are, fail out
	if !rf.otherIsRecent(lastIncludedIndex, lastIncludedTerm) {
		fmt.Printf("%d: %d: [NEW] Attempted to install stale snapshot\n", rf.me, rf.term)
		return false
	}

	fmt.Printf("%d: %d: [NEW] Conditionally installing snapshot...\n", rf.me, rf.term)
	rf.doInstallSnapshot(lastIncludedIndex, lastIncludedTerm, snapshot, true)
	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.doInstallSnapshot(index, rf.log[index].Term, snapshot, false)
}

type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}
