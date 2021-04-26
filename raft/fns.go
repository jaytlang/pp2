package raft

import (
	"math/rand"
	"sort"
	"time"
)

// Install a snapshot and trim the log, while holding the lock
// Assumes that rf.log[index] exists
func (rf *Raft) doInstallSnapshot(index int, term int, snapshot []byte, changeCommit bool) {

	// Replace totality of the log with the snapshot by freeing head of
	// the log for garbage collection
	for i := rf.snapshotIndex; i < index; i++ {
		if len(rf.log) == i {
			rf.log = append(rf.log, nil)
		} else {
			rf.log[i] = nil
		}
	}

	// If the log isn't long enough, add a dummy index entry
	// to it. Else, modify the index entry in place to reflect
	// the right term. This has already been committed, so is only
	// used to compute lastLogTerm/Index in doHeartbeat
	if len(rf.log) == index {
		rf.log = append(rf.log, &LogEntry{Term: term})
	} else {
		rf.log[index].Term = term
	}

	// Install the snapshot proper. The log now starts from rf.snapshotIndex
	rf.snapshot = snapshot
	rf.snapshotIndex = index
	if changeCommit {
		rf.commitIdx = index
		rf.appliedIdx = index
	}
	rf.persist()
	//fmt.Printf("%d: %d: [NEW] Installed snapshot successfully. New SI is %d\n", rf.me, rf.term, index)
}

// Returns true if initial processing succeeded,
// false if not. If false is returned, we convert
// to follower and update rf.term <- msgTerm
func (rf *Raft) msgCommon(msgTerm int) bool {
	if msgTerm > rf.term {
		rf.term = msgTerm
		rf.persist()
		rf.st = follower
		return false
	}

	return true
}

func (rf *Raft) disambiguator() {
	for {
		msg := <-rf.disam
		rf.lc <- msg
	}
}

func (rf *Raft) firstIndexForTerm(term int) int {
	for i, ent := range rf.log {
		if ent != nil {
			if ent.Term == term {
				return i
			}
		}
	}

	return -1
}

func (rf *Raft) otherIsRecent(oi int, ot int) bool {
	mi := len(rf.log) - 1
	mt := rf.log[mi].Term

	if ot > mt {
		return true
	} else if ot < mt {
		return false
	} else if oi >= mi {
		return true
	} else {
		return false
	}
}

func (rf *Raft) applyOutstanding() {
	msgs := []ApplyMsg{}

	if rf.commitIdx > rf.appliedIdx {
		for i := rf.appliedIdx + 1; i <= rf.commitIdx; i++ {
			le := rf.log[i]
			a := ApplyMsg{
				CommandValid: true,
				Command:      le.Cmd,
				CommandIndex: i,
			}
			msgs = append(msgs, a)
			//fmt.Printf("%d: %d: Commiting message %d\n", rf.me, rf.term, i)
		}
	}

	rf.appliedIdx = rf.commitIdx
	for i := 0; i < len(msgs); i++ {
		rf.disam <- msgs[i]
	}
}

// Initialize leader data structures
// after an election
func (rf *Raft) leaderInit() {
	rf.atServer = make([]int, len(rf.peers))
	rf.nextForServer = make([]int, len(rf.peers))
	for i := 0; i < len(rf.nextForServer); i++ {
		rf.nextForServer[i] = len(rf.log)
	}

}

// Current UNIX time in ms
func mkTimeMs() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// Uses the intervals from raft.go
func genInterval() int {
	return rand.Intn(maxWait-minWait) + minWait
}

// Returns number of units slept in ms
func sleepUntilTimeout() int {
	interval := genInterval()
	time.Sleep(time.Duration(interval) * time.Millisecond)
	return interval
}

// Vote counter
func (vc *voteCounter) countVote() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.votes++
}

// Returns -1 if is killed
func (vc *voteCounter) getVotes() int {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if vc.killed {
		return -1
	}

	return vc.votes
}

func (vc *voteCounter) killVote() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.killed = true
}

// Send the ballots and count the votes
func (rf *Raft) runElection() {
	rf.mu.Lock()

	// Set up the vote counter and the ballot
	vc := new(voteCounter)
	rva := RequestVoteArgs{
		Term:         rf.term,
		CandidateId:  rf.me,
		LastLogIndex: len(rf.log) - 1,
		LastLogTerm:  rf.log[len(rf.log)-1].Term,
	}

	//fmt.Printf("%d: %d: Running election...\n", rf.me, rf.term)

	for i := range rf.peers {
		if i == rf.me {
			// Count our own vote and carry on
			vc.countVote()
			continue
		}

		go func(j int) {
			// Send a vote. If the server is ded or
			// partitioned from us, just jump out
			// We do NOT hold the lock. Trying this first
			rvr := new(RequestVoteReply)
			result := rf.doRV(j, &rva, rvr)
			if result != nil {
				return
			}

			rf.mu.Lock()
			// If we just converted to follower,
			// that's a MASSIVE F in the chat. Tell
			// the coordinating thread we done here
			if !rf.msgCommon(rvr.Term) {
				//fmt.Printf("%d: %d: Vote killed\n", rf.me, rf.term)
				vc.killVote()
				rf.mu.Unlock()
				return
			}

			rf.mu.Unlock()

			// So we're still a candidate and the
			// term matches up. Count the vote if it is for us
			if rvr.VoteGranted {
				vc.countVote()
			}
		}(i)
	}

	// Meanwhile, we the main routine need to release the
	// lock because msgCommon is going to acquire it. As we
	// do that, we need to start the election timer by periodically
	// checking against a known end time whenever we acquire.
	rf.mu.Unlock()
	expiryTime := mkTimeMs() + int64(genInterval())

	// Let's start getting results!
	for expiryTime > mkTimeMs() {
		rf.mu.Lock()

		// Are we now suddenly a follower? ):
		voteCount := vc.getVotes()
		if voteCount < 0 || rf.st == follower {
			//fmt.Printf("%d: %d: Election attempt: became a follower\n", rf.me, rf.term)
			rf.mu.Unlock()
			return
		}

		// OK, so we aren't a follower, which is convenient.
		// Do we have enough votes to become the leader though?
		// If so, tell beCandidate that they're done being a candidate
		if voteCount > len(rf.peers)/2 {
			//fmt.Printf("%d: %d: Election attempt: became leader with %d votes\n", rf.me, rf.term, voteCount)
			rf.st = leader
			rf.leaderInit()
			rf.mu.Unlock()
			return
		}

		// If none of the above are true, we don't have the
		// votes yet but election time hasn't expired

		rf.mu.Unlock()
		time.Sleep(1 * time.Millisecond)
	}

	// If we're here, election time has expired
	// rf.st will still read candidate, so beCandidate
	// can re-issue an election. Debug statements require
	// re-acquisition of the lock
	//fmt.Printf("%d: %d: Election runtime expired\n", rf.me, rf.term)
}

// Running an election
// Returns true if we are the new leader,
// false otherwise. This encompasses the entire
// candidate lifecycle
func (rf *Raft) beCandidate() bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	//fmt.Printf("%d: %d: Entering candidacy\n", rf.me, rf.term)

	// Initialize candidacy
	rf.st = candidate

	// Run the election!
	for rf.st == candidate && !rf.killed() {
		// Vote for ourselves
		rf.votedFor = rf.me

		// Increment term and persist state
		rf.term++
		rf.persist()

		// Send RPCs
		rf.mu.Unlock()
		rf.runElection()
		time.Sleep(10 * time.Millisecond)
		rf.mu.Lock()
	}

	// If we're here, the election
	// has a decisive outcome. What is it?
	//fmt.Printf("%d: %d: exiting candidacy\n", rf.me, rf.term)
	if rf.st == leader {
		//fmt.Printf("RAFT: New leader: %d/%d\n", rf.me, rf.term)
		go rf.rvwCommits()
	}
	return rf.st == leader
}

// Leader: send heartbeats to
// EVERYONE
func (rf *Raft) doHeartbeat() {

	// Set up the heartbeat structure and start iteration,
	// again assuming that we aren't needing to hold the lock
	// to doAE. We will let these goroutines send the heartbeats,
	// and then they'll acquire/demote us to follower if needed.
	// The running timer loop should figure this out when it's time.

	//fmt.Printf("%d: %d: leader is heartbeating with nfi/as %v, %v\n", rf.me, rf.term, rf.nextForServer, rf.atServer)
	rf.ts = mkTimeMs()

	for i := range rf.peers {
		if i == rf.me {
			rf.atServer[rf.me] = len(rf.log) - 1
			rf.nextForServer[rf.me] = len(rf.log)
			continue
		}

		// We need lli. If lli doesn't exist for us, we need
		// to send an InstallSnapshot instead of appendEntries
		lli := rf.nextForServer[i] - 1
		if rf.log[lli] != nil {
			ents := make([]LogEntry, len(rf.log)-lli-1)
			for i, e := range rf.log[lli+1:] {
				ents[i] = *e
			}

			aea := AppendEntriesArgs{
				Term:            rf.term,
				LeaderId:        rf.me,
				LastLogIndex:    lli,
				LastLogTerm:     rf.log[lli].Term,
				LeaderCommitIdx: rf.commitIdx,
				Entries:         ents,
			}

			go func(j int) {
				// Send our heartbeat, and if the
				// server is dead, ignore the result
				aer := new(AppendEntriesReply)

				// Blindly post-process aer, which may
				// demote us to follower but this routine
				// does not care
				result := rf.doAE(j, &aea, aer)
				if result != nil {
					return
				}

				rf.mu.Lock()
				if !rf.msgCommon(aer.Term) {
					//fmt.Printf("%d: %d: Note: leader is now follower\n", rf.me, rf.term)
				} else if !aer.Success {
					if aer.XTerm > 0 {
						if ni := rf.firstIndexForTerm(aer.XTerm); ni > 0 {
							// Case 2: leader has XTerm
							//fmt.Printf("%d: %d: Follower %d wants term %d, we have it at %d of %d\n", rf.me, rf.term, j, aer.XTerm, ni, len(rf.log)-1)
							rf.nextForServer[j] = ni
						} else {
							// Case 1: leader doesn't have XTerm
							//fmt.Printf("%d: %d: Follower %d requests index %d for start of term %d, we have %d\n", rf.me, rf.term, j, aer.XIndex, aer.XTerm, len(rf.log)-1)
							rf.nextForServer[j] = aer.XIndex
						}
					} else if aer.XLen > 0 {
						// Case 3: follower's log is too short
						//fmt.Printf("%d: %d: Follower %d log length %d too short, we have %d\n", rf.me, rf.term, j, aer.XLen, len(rf.log)-1)
						rf.nextForServer[j] = aer.XLen
					}
				} else {
					rf.atServer[j] = aea.LastLogIndex + len(aea.Entries)
					rf.nextForServer[j] = rf.atServer[j] + 1
				}
				rf.mu.Unlock()
			}(i)

		} else {
			//fmt.Printf("%d: %d: [NEW] Don't have entry %d for follower %d, sending snapshot\n", rf.me, rf.term, lli, i)
			// Send InstallSnapshot instead
			isa := InstallSnapshotArgs{
				Term:              rf.term,
				LeaderId:          rf.me,
				LastIncludedIndex: rf.snapshotIndex,
				LastIncludedTerm:  rf.log[rf.snapshotIndex].Term,
				Data:              rf.snapshot,
			}

			go func(j int) {
				// INTERRUPT SERVICE ROUTINE
				isr := new(InstallSnapshotReply)
				result := rf.doIS(j, &isa, isr)
				if result != nil {
					return
				}
				rf.mu.Lock()

				if !rf.msgCommon(isr.Term) {
					//fmt.Printf("%d: %d: Note: leader is now follower\n", rf.me, rf.term)
				} else {
					rf.nextForServer[j] = isa.LastIncludedIndex + 1
					//fmt.Printf("%d: %d: [NEW] Snapshot sent successfully to %d, next for them is %d\n", rf.me, rf.term, j, rf.nextForServer[j])
				}
				rf.mu.Unlock()
			}(i)
		}
	}
}

func (rf *Raft) rvwCommits() {
	for {
		time.Sleep(10 * time.Millisecond)
		rf.mu.Lock()

		if rf.st != leader {
			rf.mu.Unlock()
			return
		}

		sorted := make([]int, len(rf.atServer))
		copy(sorted, rf.atServer)
		sort.Ints(sorted)

		median := sorted[(len(sorted)-1)/2]
		if len(sorted)%2 == 0 {
			median = sorted[len(sorted)/2]
		}

		if median > rf.commitIdx && rf.term == rf.log[median].Term {
			//fmt.Printf("%d: %d: leader current median up to %d from %v\n", rf.me, rf.term, median, rf.atServer)
			rf.commitIdx = median
			rf.applyOutstanding()
		}
		rf.mu.Unlock()
	}

}
