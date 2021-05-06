package inode

import (
	"errors"
	"log"
	"pp2/bio"
	"pp2/jrnl"
	"pp2/labgob"
)

const dirDataBlks = 12
const inDirDataBlks = bio.BlockSize / 8
const firstBlkAddr = jrnl.EndJrnl + 1
const numInodes = 16384
const rootInum = 0

const EndInode = firstBlkAddr + numInodes + 1

// maximum filesize is 2.04 mb ((dirdatablks+indirdatablks)*4096 bytes)

type IType byte

const (
	Dir IType = iota
	File
)

type Inode struct {
	Serialnum uint
	Refcnt    uint
	Filesize  uint
	Addrs     []uint
	Mode      IType
	// timestamp Time
}

type DirEnt struct {
	Filename string
	Inodenum uint16
}

var rootDirEnt = &DirEnt{
	Filename: "/",
	Inodenum: rootInum,
}

// Always succeeds, might take awhile
func Alloci(t *jrnl.TxnHandle, mode IType) *Inode {
retry:
	for i := firstBlkAddr; i < firstBlkAddr+numInodes; i++ {
		blk := bio.Bget(uint(i))
		if blk.Data == "" {
			ni := &Inode{
				Serialnum: uint(i) - firstBlkAddr,
				Refcnt:    1,
				Filesize:  0,
				Addrs:     []uint{},
				Mode:      mode,
			}
			blk.Data = ni.Encode()
			if t.WriteBlock(blk) != nil {
				goto retry
			}
			return ni

		}
		ni := IDecode(blk.Data)
		if ni.Refcnt == 0 {
			ni = &Inode{
				Serialnum: uint(i) - firstBlkAddr,
				Refcnt:    1,
				Filesize:  0,
				Addrs:     []uint{},
				Mode:      mode,
			}
			blk.Data = ni.Encode()
			if t.WriteBlock(blk) != nil {
				goto retry
			}
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
// Then, relse. The decrement may fail.
func (i *Inode) Free(t *jrnl.TxnHandle) error {
	if i.Refcnt == 0 {
		log.Fatal("double free")
	}

	i.Refcnt--
	b := &bio.Block{
		Nr:   i.Serialnum + firstBlkAddr,
		Data: i.Encode(),
	}
	if err := t.WriteBlock(b); err != nil {
		return err
	}
	i.Relse()
	return nil
}

// May fail silently (implicit success)
func (i *Inode) Relse() {
	actual := i.Serialnum + firstBlkAddr
	b := &bio.Block{
		Nr:   actual,
		Data: i.Encode(),
	}
	b.Brelse()
}

// Always succeeds
func Geti(id uint) *Inode {
	id = firstBlkAddr + id
	if id >= firstBlkAddr+numInodes {
		log.Fatal("inode id out of range")
	}

	blk := bio.Bget(id)
	if blk.Data == "" {
		log.Fatal("empty Inode")
	}
	ni := IDecode(blk.Data)
	return ni
}

// May fail if we've lost the lock by this point
func (i *Inode) Renew() error {
	b := &bio.Block{
		Nr:   i.Serialnum + firstBlkAddr,
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
	labgob.Register(&DirEnt{})
}
