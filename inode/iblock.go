package inode

import (
	"errors"
	"fmt"
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
// Enqueues inode changes for writing
func (i *Inode) increaseSize(t *jrnl.TxnHandle, ns uint) error {
	fmt.Printf("Increasing size of inode w/ serial num %d\n", i.Serialnum)
	if i.Filesize >= ns {
		log.Fatal("unneeded alloc")
	} else if saneCeil(ns, 4096) > nDirectBlocks {
		return errors.New("file would be too large")
	}

	currentBlocks := saneCeil(i.Filesize, 4096)
	newBlocks := saneCeil(ns, 4096)
	addedBlocks := newBlocks - currentBlocks

	if addedBlocks > 0 {
		fmt.Printf("c: %d vs. n: %d vs. a: %d\n", currentBlocks, newBlocks, addedBlocks)
		fmt.Printf("Old iaddrs length: %d\n", len(i.Addrs))
		blnl := balloc.AllocBlocks(t, addedBlocks)
		i.Addrs = append(i.Addrs, blnl...)
	}
	i.Filesize = ns
	i.EnqWrite(t)
	fmt.Printf("New i.Addrs length enqueued: %d\n", len(i.Addrs))
	return nil
}

// Decreases filesize to zero
// Burns the call into balloc since it frees everything
// Enqueues inode changes for writing
// Does not fail
func (i *Inode) truncate(t *jrnl.TxnHandle) {
	fmt.Printf("Truncating inode w/ serial num %d\n", i.Serialnum)
	// Free every single block
	balloc.RelseBlocks(t, i.Addrs)
	i.Addrs = []uint{}
	i.Filesize = 0
	i.EnqWrite(t)
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
	defer i.Relse()
	res := ""

	fmt.Printf("Reading %d bytes from inode w/ serial num %d\n", count, i.Serialnum)

	// Setup the first block
	bn := offset / 4096
	bo := offset % 4096

	// Check that the first block exists
	if bn >= uint(len(i.Addrs)) {
		fmt.Printf("Note: bn > len(i.Addrs)\n")
		return ""
	}

	for j := bn; count > 0; j++ {
		blk := bio.Bget(i.Addrs[j])
		data := blk.Data
		fmt.Printf("Block data: %s\n", data)

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
		fmt.Printf("count: %d\n", count)

		// Read the data
		rb := []byte(res)
		rb = append(rb, []byte(data[:toread])...)
		res = string(rb)

		// Get the next block. If there isn't one,
		// return as we are
		blk.Brelse()
		if j == uint(len(i.Addrs))-1 {
			return res
		}
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
	defer i.Relse()

	fmt.Printf("Writing inode w/ serial num %d\n", i.Serialnum)
	// Setup the first block
	bn := offset / 4096
	bo := offset % 4096

	// Check that the first block exists
	// == is ok if the last block takes up all 4096 bytes or whatever
	fmt.Printf("bn: %d, bo: %d\n", bn, bo)
	if bn >= nDirectBlocks {
		return 0, errors.New("maximum valid blocks exceeded")
	} else if offset > i.Filesize && (offset != 0 && i.Filesize != 0) {
		return 0, errors.New("tried to append past the end of the file")
	}

	totalbytes := uint(len([]byte(data)))
	tb := totalbytes

	if offset+tb > nDirectBlocks*bio.BlockSize {
		return 0, errors.New("that write too big")
	}
	if offset+tb > i.Filesize {
		i.increaseSize(t, offset+tb)
	}

	for j := bn; totalbytes > 0; j++ {
		blk := bio.Bget(i.Addrs[j])
		bdata := blk.Data

		// If the block offset is > 0, regardless of data's length...
		// Leave bdata up to bo standing, trim data appropriately
		if bo > 0 {
			// We've already checked that the offset doesn't
			// go over the end of the file. Therefore, bdata[:bo]
			// will work out no matter what
			// If len(data) <= 4096 - bo, write all of data (CHECKME)
			// Otherwise, write data[:4096-bo]
			if totalbytes > 4096-bo {
				bdata = bdata[:bo] + data[:4096-bo]
				data = data[4096-bo:]
				totalbytes -= (4096 - bo)
			} else {
				if uint(len(bdata)) <= bo+uint(len(data)) {
					bdata = bdata[:bo] + data
				} else {
					bdata = bdata[:bo] + data + bdata[uint(len(data))+bo:]
				}
				totalbytes = 0
				// Will break, don't change data
			}

			// Reset the block offset
			bo = 0

		} else if totalbytes < 4096 {
			// We are towards the end of the data. Write what's left
			// of data (data[:totalbytes]) and leave bdata[totalbytes:]
			// if the bdata is long enough
			if len(bdata) > int(totalbytes) {
				bdata = data[:totalbytes] + bdata[totalbytes:]
			} else {
				bdata = data[:totalbytes] + bdata
			}

			// Cause the break, don't touch data cuz we don't need to
			totalbytes = 0
		} else {
			// We are not towards the end of the data. Overwrite the block
			// and trim the data
			bdata = data[:4096]
			totalbytes -= 4096
			data = data[4096:]
		}

		blk.Data = bdata
		t.WriteBlock(blk)
		blk.Brelse()
	}

	return tb, nil
}
