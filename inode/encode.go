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
