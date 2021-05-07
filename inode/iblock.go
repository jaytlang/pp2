package inode

import (
	"errors"
	"log"
	"math"
	"pp2/balloc"
	"pp2/bio"
	"pp2/jrnl"
)

// Pain
func saneCeil(a uint, b uint) uint {
	return uint(math.Ceil(float64(a) / float64(b)))
}

// Increases filesize to ns
// Fails if ns <= i.Filesize, errors if
// filesize will exceed dataBlks
// Burns the allocblocks call
// May fail if allocblocks fails e.g. we get
// cut off before bitmapget releases for some reason
// Enqueues inode changes for writing
func (i *Inode) increaseSize(t *jrnl.TxnHandle, ns uint) error {
	if i.Filesize >= ns {
		log.Fatal("unneeded alloc")
	} else if saneCeil(ns, 4096) > dataBlks {
		return errors.New("file would be too large")
	}

	currentBlocks := saneCeil(i.Filesize, 4096)
	newBlocks := saneCeil(ns, 4096)
	addedBlocks := newBlocks - currentBlocks

	if addedBlocks > 0 {
		blnl, err := balloc.AllocBlocks(t, addedBlocks)
		if err != nil {
			return err
		}

		// Go through the inode and update the directs
		i.Addrs = append(i.Addrs, blnl...)
	}
	i.Filesize = ns
	i.EnqWrite(t)

	return nil
}

// Decreases filesize to zero
// Burns the call into balloc since it frees everything
// May fail if we drop offline in the middle of it
// Enqueues inode changes for writing
func (i *Inode) truncate(t *jrnl.TxnHandle) error {
	// Free every single block
	err := balloc.RelseBlocks(t, i.Addrs)
	if err != nil {
		return err
	}
	i.Addrs = []uint{}
	i.Filesize = 0
	i.EnqWrite(t)
	return nil
}

// Reads a certain count of data from a certain
// offset within an inode.
// Doesn't burn any balloc calls, makes no inode changes
// Releases every block it touches without modifying it
// In this sense, guaranteed to succeed
func Readi(inum uint16, offset uint, count uint) string {
	// Get the inode in question
	// Panics if this fails
	i := Geti(inum)
	res := ""

	// Setup the first block
	bn := saneCeil(offset, 4096)
	bo := offset % 4096

	// Check that the first block exists
	if bn >= uint(len(i.Addrs)) {
		return ""
	}

	for j := bn; count > 0; j++ {
		blk := bio.Bget(i.Addrs[bn])
		data := blk.Data

		// First iteration only: block offset check
		if bo > 0 {
			// If the block offset is off the end, fail
			if bo >= uint(len(data)) {
				blk.Brelse()
				return res
			}
			data = data[bo:]
			bo = 0
		}

		// Deduct from count
		toread := uint(len(data))
		if count < toread {
			toread = count
		}
		count -= toread

		// Read the data
		rb := []byte(res)
		rb = append(rb, []byte(data)...)
		res = string(rb)

		// Get the next block. If there isn't one,
		// return as we are
		if j == uint(len(i.Addrs))-1 {
			blk.Brelse()
			return res
		}
		blk.Brelse()
	}
	return res
}

// The same as readi, with a few exceptions:
// - writes that start at or cross the end of
// the file grow the file, up to the maximum file siz
// - loop copies data into buffers obviously, then buffers are enqueued
// into log (BUT NOT WRITTEN THROUGH!!)
func Writei(t *jrnl.TxnHandle, inum uint16, offset uint, data string) (uint, error) {
	// Get the inode in question
	// Panics if this fails
	i := Geti(inum)

	// Setup the first block
	bn := saneCeil(offset, 4096)
	bo := offset % 4096

	// Check that the first block exists
	// == is ok if the last block takes up all 4096 bytes or whatever
	if bn >= dataBlks {
		return 0, errors.New("maximum valid blocks exceeded")
	} else if offset > i.Filesize+1 {
		return 0, errors.New("tried to append past the end of the file")
	}

	totalbytes := uint(len([]byte(data)))

	for j := bn; totalbytes > 0; j++ {
		blk := bio.Bget(i.Addrs[bn])
		bdata := blk.Data

		if bo > 0 {
			bdata = bdata[:bo] + data[:4096-bo]
			bo = 0
			totalbytes -= (4096 - bo)
		} else if totalbytes < 4096 {
			bdata = data[:totalbytes] + bdata[totalbytes:]
			totalbytes = 0
		} else {
			bdata = data[:4096]
			totalbytes -= 4096
		}

		blk.Data = bdata
		t.WriteBlock(blk)
		blk.Brelse()
	}
	return totalbytes, nil
}
