package inode

import (
	"errors"
	"fmt"
	"log"
	"pp2/bio"
	"pp2/jrnl"
	"pp2/labgob"
)

const nDirectBlocks = 511
const firstInodeAddr = jrnl.EndJrnl + 2
const numInodes = 16384
const RootInum = 0

const EndInode = firstInodeAddr + numInodes + 1

type IType byte

const (
	Dir IType = iota
	File
)

type Inode struct {
	Serialnum uint16
	Refcnt    uint16
	Filesize  uint
	Addrs     []uint
	Mode      IType
	// timestamp Time
}

// Always succeeds, might take awhile
func Alloci(t *jrnl.TxnHandle, mode IType) *Inode {
retry:
	for i := firstInodeAddr; i < firstInodeAddr+numInodes; i++ {
		blk := bio.Bget(uint(i))
		if blk.Data == "" {
			ni := &Inode{
				Serialnum: uint16(i - firstInodeAddr),
				Refcnt:    1,
				Addrs:     []uint{},
				Mode:      mode,
			}
			if ni.EnqWrite(t) != nil {
				goto retry
			}
			fmt.Printf("Acquired inode w/ serial num %d from empty\n", ni.Serialnum)
			return ni

		}
		ni := IDecode(blk.Data)
		if ni.Refcnt == 0 {
			ni = &Inode{
				Serialnum: uint16(i - firstInodeAddr),
				Refcnt:    1,
				Addrs:     []uint{},
				Mode:      mode,
			}
			if ni.EnqWrite(t) != nil {
				goto retry
			}
			fmt.Printf("Acquired inode w/ serial num %d from non-empty, refcnt %d\n", ni.Serialnum, ni.Refcnt)
			return ni
		}
		blk.Brelse()
	}
	log.Fatal("no allocatable Inodes")
	// Never reached
	return nil
}

// Decrement the refcount on the inode. If it
// hits zero, further allocs might pick it up.
// Then, relse. The decrement may fail if blkPerSys
// is exceeded, but this is unlikely
func (i *Inode) Free(t *jrnl.TxnHandle) error {
	if i.Refcnt == 0 {
		log.Fatal("double free")
	}

	i.Refcnt--
	if err := i.EnqWrite(t); err != nil {
		return err
	}
	i.Relse()
	fmt.Printf("Freed inode w/ serial num %d, refcnt %d\n", i.Serialnum, i.Refcnt)
	return nil
}

// May fail silently (implicit success)
func (i *Inode) Relse() {
	actual := uint(i.Serialnum) + firstInodeAddr
	b := &bio.Block{
		Nr:   actual,
		Data: i.Encode(),
	}
	b.Brelse()
	fmt.Printf("Released inode w/ serial num %d\n", i.Serialnum)
}

// Always succeeds
// Panics if the inode doesn't exist
func Geti(inum uint16) *Inode {
	id := firstInodeAddr + uint(inum)
	if id >= firstInodeAddr+numInodes {
		log.Fatal("inode id out of range")
	}

	blk := bio.Bget(id)
	if blk.Data == "" {
		log.Fatal("empty Inode")
	}
	ni := IDecode(blk.Data)
	fmt.Printf("Got inode w/ serial num %d, refcnt %d\n", ni.Serialnum, ni.Refcnt)
	return ni
}

// Update the inode in place without doing
// anything else. May fail if blkPerSys is exceeded,
// but otherwise will succeed okay
// Note that changes don't write through immediately
func (i *Inode) EnqWrite(t *jrnl.TxnHandle) error {
	b := &bio.Block{
		Nr:   uint(i.Serialnum) + firstInodeAddr,
		Data: i.Encode(),
	}
	if err := t.WriteBlock(b); err != nil {
		return err
	}
	fmt.Printf("Enqueued inode to write w/ serial num %d, refcnt %d\n", i.Serialnum, i.Refcnt)
	return nil
}

// May fail if we've lost the lock by this point
func (i *Inode) Renew() error {
	b := &bio.Block{
		Nr:   uint(i.Serialnum) + firstInodeAddr,
		Data: i.Encode(),
	}
	err := b.Brenew()
	if err != bio.OK {
		return errors.New("failed to renew lock")
	}
	return nil
}

func InodeInit() {
	labgob.Register(&Inode{})
}
