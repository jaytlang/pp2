package jrnl

import (
	"errors"
	"fmt"
	"pp2/bio"
)

func resurrect(sb *logSB) error {
	// Called if sb.commit = 1
	// Shares code with commit()

	fmt.Printf("Resurrecting log\n")

	err := replay(sb)
	if err != nil {
		return err
	}
	sb.commit = 0
	sb.bitmap = ""
	sb.cnt = 0

	for i := 0; i < sysPerLog; i++ {
		sb.bitmap += "0"
	}
	berr := flattenSb(sb).Bpush()
	if berr != bio.OK {
		return errors.New("lock lease expired")
	}
	return nil
}

func commit(sb *logSB) error {
	// LET THE MAGIC HAPPEN
	sb.commit = 1
	err := flattenSb(sb).Bpush()
	if err != bio.OK {
		// We did NOT commit our transaction
		// It is entirely possible that another person
		// performed the commit for us. Return error
		return errors.New("lock lease expired")
	}
	fmt.Printf("committed\n")

	// Now, we replay the log to disk.
	// Note that the SB lock HAS to be held
	// during this time!!
	rerr := replay(sb)
	if rerr != nil {
		// We lost the superblock halfway through.
		// However, data is still committed, possibly
		// Return for the caller to check this
		return errors.New("lock lease expired")
	}

	// Once we are replayed, indicate that we are
	// no longer committed and blow away the sb
	sb.commit = 0

	// The sb will now be blown away if there is a crash
	// Reset for continued use
	sb.bitmap = ""
	for i := 0; i < sysPerLog; i++ {
		sb.bitmap += "0"
	}

	err = flattenSb(sb).Bpush()
	if err != bio.OK {
		return errors.New("lock lease expired")
	}

	// Returns for the held SB to be released
	return nil
}

func replay(sb *logSB) error {
	// Replay every valid log segment
	for i, v := range sb.bitmap {
		if v == '1' {
			err := replayLogSegment(sb, uint(i))
			// Error is nil -> we lost the superblock!
			if err != nil {
				return errors.New("lock lease expired")
			}
		}
	}
	return nil
}

func replayLogSegment(sb *logSB, sgmt uint) error {
	lbn := getLogSegmentStart(sgmt)
	fmt.Printf("Replaying block segment %d to disk\n", sgmt)
	for {
	retry:
		lb := parseLb(bio.Bget(lbn))

		// Somebody else might *get* this block,
		// but nobody else will be able to write to it
		// because they don't hold the lock on the sb
		// and if the sb lock was lost, the log would
		// replay anyway. The loss of this lock either implies
		//
		// -> somebody else wanted to do a read. That's fine, we can
		// take it back from them at our leisure and update it.
		//
		// -> we lost the sb lock and someone else is committing
		// over us simultaneously, e.g. we are racing. Brenew()
		// will thus fail when we exit out.
		//
		// Solution: if this block fails, set a flag. If brenew() comes
		// back badly, we can assume the sb lock is gone. Otherwise, we
		// try again.
		db := &bio.Block{
			Nr:   lb.rnr,
			Data: lb.rdata,
		}

		flattenLb(lb).Brelse()
		err := db.Bpush()
		berr := flattenSb(sb).Brenew()
		if berr != bio.OK {
			return errors.New("lost the superblock lock")
		} else if err != bio.OK {
			bio.Bget(lb.rnr)
			goto retry
		}

		if lb.last {
			break
		}
		lbn++
	}
	return nil
}
