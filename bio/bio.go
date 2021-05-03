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

// const blockSize = 4096

const (
	OK BioError = iota
	ErrNoLock
	ErrBadSize
)

var dsk *kvraft.Clerk

func Binit() {
	conf := netdrv.MkDefaultNetConfig(false)
	dsk = kvraft.MakeClerk(conf)
}

// Acquires a block along with its
// lock. Will continually contend for
// a given lock until it gets it, then
// return back.
func Bget(nr uint) *Block {
	nstr := fmt.Sprintf("%d", nr)

retry:
	dsk.Acquire(nstr)
	data, err := dsk.ReadFromFile(nstr)
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

	/*
		if len(b.Data) > blockSize {
			return ErrBadSize
		}
	*/

	err := dsk.WriteToFile(nstr, b.Data)
	if err != nil {
		return ErrNoLock
	}
	return OK
}

func (b *Block) Brenew() BioError {
	nstr := fmt.Sprintf("%d", b.Nr)
	err := dsk.Renew(nstr)
	if err != nil {
		log.Print("warning: renew called without lock")
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
