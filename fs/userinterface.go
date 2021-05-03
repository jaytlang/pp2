package fs

import (
	"pp2/inode"
)

type Filesystem interface {
	Create(filename string, mode inode.IType) int
	Open(filename string) int
	Read(fd int, count uint) string
	Write(fd int, data string) uint
	Close(fd int) int
	Remove(filename string)
}

type PigeonFs struct {
	fdTbl map[int]inode.DirEnt
}

/*
func (p *PigeonFs) Create(filename string, mode inode.IType) int {
	parent := filename
	if filename != "/" {
		parent = path.Dir(filename)
	}
	i := inode.Alloci(mode)
}
*/
