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
const inodeNum = 16384
const rootInum = 0

const EndInode = firstBlkAddr + inodeNum + 1

// maximum filesize is 2.04 mb ((dirdatablks+indirdatablks)*4096 bytes)

type IType byte

const (
	Dir IType = iota
	File
)

type Inode struct {
	Serialnum uint
	Refcnt    uint
	Filesize  uint // # blocks, not bytes
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

func Alloci(mode IType) *Inode {
	for i := firstBlkAddr; i < firstBlkAddr+inodeNum; i++ {
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
			// jrnl.AtomicWrite([]*bio.Block{blk})
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
			// jrnl.AtomicWrite([]*bio.Block{blk})
			return ni
		}
	}
	log.Fatal("no allocatable Inodes")
	return nil
}

func (i *Inode) Relse() {
	actual := i.Serialnum + firstBlkAddr
	b := &bio.Block{
		Nr:   actual,
		Data: i.Encode(),
	}
	b.Brelse()
}

func Geti(id uint) *Inode {
	if firstBlkAddr < id || id <= firstBlkAddr+inodeNum {
		log.Fatal("inode id out of range")
	}

	blk := bio.Bget(id)
	if blk.Data == "" {
		log.Fatal("empty Inode")
	} else {
		ni := IDecode(blk.Data)
		return ni
	}

	return nil
}

/*

func (i *Inode) Free() {
	if i.Refcnt == 0 {
		log.Fatal("double free")
	}

	i.Refcnt--
	b := &bio.Block{
		Nr:   i.Serialnum + firstBlkAddr,
		Data: i.Encode(),
	}
	jrnl.AtomicWrite([]*bio.Block{b})
	i.Relse()
}

*/
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
