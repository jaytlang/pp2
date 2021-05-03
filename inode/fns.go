package inode

import (
	"errors"
	"path"
	"pp2/bio"
	"strings"
)

/*
func (p *PigeonFs) walkToDir(dir string) (*inode.DirEnt, error) {
	dir = path.Clean(dir)
	els := strings.Split(dir, "/")
	d := &inode.DirEnt{
		Filename: "/",

		Inodenum: inode.RootInum,
	}

	for _, el := range els {
		i := inode.Geti(uint(d.Inodenum))
		if i.Mode != inode.Dir {
			return nil, errors.New("target is not a directory")
		}
		for _, a := range i.Addrs {
			b := bio.Bget(a)
			de := inode.DDecode(b.Data)
			if de.Filename == el {

			}
		}

	}
}
*/

func Lookup(dir string) (*DirEnt, error) {
	dir = path.Clean(dir)
	d := *rootDirEnt

	if dir == "/" {
		return &d, nil
	}

	els := strings.Split(dir, "/")
	for idx, el := range els {
		i := Geti(uint(d.Inodenum))
		if i.Mode != Dir && idx != len(els)-1 {
			return nil, errors.New("can't walk file")
		}

		for j := 0; uint(j) < i.Filesize; j++ {
			blk := i.getAddrNo(uint(j))
			nd := DDecode(blk.Data)
			blk.Brelse()
			if nd.Filename == el {
				d = *nd
				continue
			}
		}

		return nil, errors.New("no such file or directory")
	}

	return &d, nil
}

func (i *Inode) getAddrNo(n uint) *bio.Block {
	// addr := -1
	// for idx, potAddr := range i.Addrs {
	// 	if idx == 13 {
	// 		indirblk := bio.Bget(uint(potAddr))
	// 	}
	// 	if potAddr == n {
	// 		addr = int(potAddr)
	// 	}
	// }

	// if addr == -1 {
	// 	log.Fatal("Address not found in inode")
	// }

	// blk := bio.Bget(uint(addr))
	// return blk
	return nil
}
