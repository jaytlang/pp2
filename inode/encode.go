package inode

import (
	"bytes"
	"pp2/labgob"
)

func (i *Inode) Encode() string {
	b := bytes.Buffer{}
	e := labgob.NewEncoder(&b)
	e.Encode(i)
	return b.String()
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
	return b.String()
}

func DDecode(blkData string) *DirEnt {
	s := new(DirEnt)
	b := bytes.NewBuffer([]byte(blkData))
	dec := labgob.NewDecoder(b)
	dec.Decode(s)
	return s
}

func (i *indirect) encode() string {
	b := bytes.Buffer{}
	e := labgob.NewEncoder(&b)
	e.Encode(i)
	return b.String()
}

func indirDecode(blkData string) *indirect {
	s := new(indirect)
	b := bytes.NewBuffer([]byte(blkData))
	dec := labgob.NewDecoder(b)
	dec.Decode(s)
	return s
}
