package bio

import (
	"fmt"
	"log"
	"pp2/kvraft"
	"pp2/netdrv"
)

type Block struct {
	Nr   uint
	Data string
}

type BioError byte

const BlockSize = 4096

const (
	OK BioError = iota
	ErrNoLock
	ErrBadSize
)

var dsk Disk

func Binit(nsAddr string, test bool) {
	if test {
		dsk = &MockDisk{
			kv: make(map[string]string),
		}
	} else {
		conf := netdrv.MkDefaultNetConfig(false, false, nsAddr)
		dsk = kvraft.MakeClerk(conf)
	}
}

// Acquires a block along with its
// lock. Will continually contend for
// a given lock until it gets it, then
// return back. Returns the empty string
// inside the appropriately-numbered block
// if it is currently empty
func Bget(nr uint) *Block {
	nstr := fmt.Sprintf("%d", nr)

retry:
	dsk.Acquire(nstr)
	data, err := dsk.Get(nstr)
	if err != nil {
		log.Print("Warning: single operation too slow for lock lease")
		goto retry
	}
	return &Block{
		Nr:   nr,
		Data: data,
	}
}

// INVARIANT: lock must be held
// otherwise an error will be returned
func (b *Block) Bpush() BioError {
	nstr := fmt.Sprintf("%d", b.Nr)

	err := dsk.Put(nstr, b.Data)
	if err != nil {
		return ErrNoLock
	}
	return OK
}

func (b *Block) Brenew() BioError {
	nstr := fmt.Sprintf("%d", b.Nr)
	err := dsk.Renew(nstr)
	if err != nil {
		return ErrNoLock
	}
	return OK
}

func (b *Block) Brelse() BioError {
	nstr := fmt.Sprintf("%d", b.Nr)
	err := dsk.Release(nstr)
	if err != nil {
		return ErrNoLock
	}
	return OK
}
