package jrnl

import (
	"errors"
	"fmt"
	"pp2/bio"
)

// Helper
func getLogSegmentStart(blkSeg uint) uint {
	return blkSeg*blkPerSys + logStart
}

// When you're done with some writes,
// you shove them all into a list
// and send it our way. Assumed to be <blkPerSys
// in length.
//
// Can and should be called concurrently.
func AtomicWrite(blks []*bio.Block) error {
	if len(blks) > blkPerSys {
		return errors.New("too many blocks written")
	}

	blkSeg := beginTransaction()

	lbn := getLogSegmentStart(blkSeg)
	for i, blk := range blks {
		lb := parseLb(bio.Bget(lbn))

		lb.last = false
		if i == len(blks)-1 {
			lb.last = true
		}
		lb.rnr = blk.Nr
		lb.rdata = blk.Data

		nlb := flattenLb(lb)
		nlb.Bpush()
		nlb.Brelse()
		lbn++
	}

	fmt.Printf("Wrote %d blocks to log segment %d\n", len(blks), blkSeg)
	endTransaction()
	return nil
}
