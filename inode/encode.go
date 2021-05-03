package inode

import (
	"bytes"
	"pp2/labgob"
)

func (i *Inode) encode() string {
	b := bytes.Buffer{}
	e := labgob.NewEncoder(&b)
	e.Encode(i)
	return string(b.Bytes())
}

func decode(blkData string) *Inode {
	s := new(Inode)
	b := bytes.NewBuffer([]byte(blkData))
	dec := labgob.NewDecoder(b)
	dec.Decode(s)
	return s
}
