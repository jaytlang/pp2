package inode

import (
	"errors"
	"log"
	"pp2/bio"
)

// Indirects are alloced/dealloced through
// the balloc API, but here's their datatype
type indirect []uint

// Always succeeds
func (i *Inode) getIndirectBlk() *indirect {
	if len(i.Addrs) <= dirDataBlks {
		log.Fatal("asked for non-existent indirect")
	}

	addr := i.Addrs[dirDataBlks]
	blk := bio.Bget(addr)
	return indirDecode(blk.Data)
}

// May fail if the lock on the indirect block
// has been lost
func (i *Inode) putIndirect(ind *indirect) error {
	if len(i.Addrs) <= dirDataBlks {
		log.Fatal("putting nonexistent indirect")
	} else if len(*ind) > inDirDataBlks {
		log.Fatal("too many indirect data blocks!")
	}

	nb := &bio.Block{
		Nr:   i.Addrs[dirDataBlks],
		Data: ind.encode(),
	}

	if nb.Bpush() != bio.OK {
		return errors.New("Failed to push block, lock lost")
	}
	return nil
}

// Only use if the indirect was not modified,
// or was already put successfully
// Will always succeed (may fail silently)
func (i *Inode) relseIndirect() {
	if len(i.Addrs) <= dirDataBlks {
		log.Fatal("asked for non-existent indirect")
	}

	nb := &bio.Block{
		Nr:   i.Addrs[dirDataBlks],
		Data: "",
	}

	nb.Brelse()
}
