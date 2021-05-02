package bio

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"pp2/kvraft"
	"pp2/netdrv"
	"time"
)

// Data is now a tmp file name
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
	// data (now _) is not actually data anymore
	_, err := dsk.Get(nstr)
	if err != nil {
		log.Print("Warning: single operation too slow for lock lease")
		goto retry
	}
	// XXX: API for blocks should change when this happens
	dbytes, err := ioutil.ReadFile(nstr)
	if err != nil {
    fmt.Println("Fail to read block from file: creating new block")
    _, err := os.Create(nstr)
    if err != nil {
      panic("Fail to create empty file")
    }
    return &Block {
      Nr: nr,
      Data: "",
    }
	}
	data := fmt.Sprint(string(dbytes))

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

	asbytes := []byte(b.Data)
	err := ioutil.WriteFile(nstr, asbytes, 0777)
	if err != nil {
		fmt.Println("Fail to write block to file")
		panic(err)
	}

	// XXX: API for blocks should change when this happens
	// Right now I am just doing timestamps for this
	err = dsk.Put(nstr, fmt.Sprintf("%d", time.Now().Unix()))
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
