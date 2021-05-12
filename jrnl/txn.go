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

type TxnHandle struct {
	blkSeg uint
	offset uint
}

// Attempt to write a block to the log.
// Semantics: will succeed unless blkPerSys exceeded
// This is highly unlikely, so assume this passes okay
// It is recommended to hold all blocks you write here,
// and to keep them through the duration of your log.
func (t *TxnHandle) WriteBlock(blk *bio.Block) error {
	if t.offset >= blkPerSys {
		return errors.New("too many blocks written")
	}
	lbn := getLogSegmentStart(t.blkSeg) + t.offset

retry:
	// Acquires and releases LOG BLOCK
	ilb := parseLb(bio.Bget(lbn))
	ilb.last = false
	ilb.rnr = blk.Nr
	ilb.rdata = blk.Data

	nlb := flattenLb(ilb)
	err := nlb.Bpush()
	if err != bio.OK {
		goto retry
	}

	nlb.Brelse()

	t.offset++
	return nil
}

// Start a transaction. Updates internal
// metadata to ensure consistency and
// returns the syscall log subset in which
// this person is to write, NOT the raw
// block number. Will always succeed.
func BeginTransaction() *TxnHandle {
	var res uint
start:
	sb := parseSb(bio.Bget(sbNr))
	if sb.commit > 0 {
		flattenSb(sb).Brelse()
		goto start
	}
	for i, c := range sb.bitmap {
		if c == '0' {
			ob := []rune(sb.bitmap)
			ob[i] = '1'
			sb.bitmap = string(ob)

			res = uint(i)
			goto done
		}
	}

	// Retry if no luck, i.e. log is outta room
	flattenSb(sb).Brelse()
	goto start

done:
	sb.cnt++
	nsb := flattenSb(sb)
	err := nsb.Bpush()
	if err != bio.OK {
		goto start
	}
	nsb.Brelse()

	return &TxnHandle{
		blkSeg: res,
		offset: 0,
	}
}

// Will always succeed. Might take a while.
// However, before you call this, ensure you hold
// all blocks that you touched during the transaction.
func (t *TxnHandle) EndTransaction(abt bool) {
markLast:
	lbn := getLogSegmentStart(t.blkSeg) + t.offset - 1
	if t.offset == 0 {
		lbn++
	}
	ilb := parseLb(bio.Bget(lbn))
	ilb.last = true
	nlb := flattenLb(ilb)
	err := nlb.Bpush()
	if err != bio.OK {
		goto markLast
	}
	nlb.Brelse()

retry:
	sb := parseSb(bio.Bget(sbNr))
	if sb.commit > 0 {
		flattenSb(sb).Brelse()
		goto retry
	}
	sb.cnt--

	if sb.cnt == 0 {
		fmt.Printf("Outstanding transactions to zero.\n")

		hasValid := false
		for _, r := range sb.bitmap {
			if r != '0' {
				hasValid = true
				break
			}
		}

		if hasValid {
			fmt.Printf("At least one valid block exists, committing...")
			err := commit(sb)
			if err != nil {
				// Lost the superblock halfway through a commit.
				// We can go get it and see if it still needs committing.
				goto retry
			}
			fmt.Printf("Committed.\n")
		} else {
			err := flattenSb(sb).Bpush()
			if err != bio.OK {
				goto retry
			}
		}
	} else {
		err := flattenSb(sb).Bpush()
		if err != bio.OK {
			goto retry
		}
	}
	flattenSb(sb).Brelse()
}

// Will always succeed. Might take a while.
// However, before you call this, ensure you hold
// all blocks that you touched during the transaction.
func (t *TxnHandle) AbortTransaction() {
retry:
	sb := parseSb(bio.Bget(sbNr))
	if sb.commit > 0 {
		flattenSb(sb).Brelse()
		goto retry
	}

	ob := []rune(sb.bitmap)
	ob[t.blkSeg] = '0'
	sb.bitmap = string(ob)

	nsb := flattenSb(sb)
	err := nsb.Bpush()
	if err != bio.OK {
		goto retry
	}

	nsb.Brelse()
	t.EndTransaction(true)
}
