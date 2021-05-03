package inode

import (
	"bytes"
	"pp2/labgob"
)

func (i *Inode) Encode() string {
	b := bytes.Buffer{}
	e := labgob.NewEncoder(&b)
	e.Encode(i)
	return string(b.Bytes())
}

func IDecode(blkData string) *Inode {
	s := new(Inode)
	b := bytes.NewBuffer([]byte(blkData))
	dec := labgob.NewDecoder(b)
	dec.Decode(s)
	return s
}

func (d *DirEnt) Encode() string {
	b := bytes.Buffer{}
	e := labgob.NewEncoder(&b)
	e.Encode(d)
	return string(b.Bytes())
}

func DDecode(blkData string) *DirEnt {
	s := new(DirEnt)
	b := bytes.NewBuffer([]byte(blkData))
	dec := labgob.NewDecoder(b)
	dec.Decode(s)
	return s
}
