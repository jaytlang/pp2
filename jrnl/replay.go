package jrnl

import (
	"fmt"
	"pp2/bio"
)

func resurrect(sb *logSB) {
	// Called if sb.commit = 1
	// Shares code with commit()

	fmt.Printf("Resurrecting log\n")

	replay(sb)
	sb.commit = 0
	sb.bitmap = ""
	sb.cnt = 0

	for i := 0; i < sysPerLog; i++ {
		sb.bitmap += "0"
	}
	flattenSb(sb).Bpush()
}

func commit(sb *logSB) {
	// LET THE MAGIC HAPPEN
	sb.commit = 1
	flattenSb(sb).Bpush()
	fmt.Printf("committed\n")

	// Now, we replay the log to disk.
	// Note that the SB lock HAS to be held
	// during this time!! Might demand a refactor
	// in how we do locking, or an extension of
	// the lock lease
	replay(sb)

	// Once we are replayed, indicate that we are
	// no longer committed and blow away the sb
	sb.commit = 0

	// The sb will now be blown away if there is a crash
	// Reset for continued use
	sb.bitmap = ""
	for i := 0; i < sysPerLog; i++ {
		sb.bitmap += "0"
	}
	flattenSb(sb).Bpush()

	// Returns for the held SB to be released
}

func replay(sb *logSB) {
	// Replay every valid log segment
	for i, v := range sb.bitmap {
		if v == '1' {
			replayLogSegment(uint(i))
		}
	}
}

func replayLogSegment(sgmt uint) {
	lbn := getLogSegmentStart(sgmt)
	fmt.Printf("Replaying block segment %d to disk\n", sgmt)
	for {
		lb := parseLb(bio.Bget(lbn))

		db := bio.Bget(lb.rnr)
		db.Data = lb.rdata
		db.Bpush()
		db.Brelse()

		flattenLb(lb).Brelse()

		if lb.last {
			break
		}
		lbn++
	}
}
