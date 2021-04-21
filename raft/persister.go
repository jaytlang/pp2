package raft

//
// support for Raft and kvraft to save persistent
// Raft state (log &c) and k/v server snapshots.
//
// we will use the original persister.go to test your code for grading.
// so, while you can modify this code to help you debug, please
// test with the original before submitting.
//

import (
	"io/ioutil"
	"log"
	"os"
	"sync"
)

const raftStateFileName string = "persiststate"

type Persister struct {
	mu sync.Mutex
}

func MakePersister() *Persister {
	return &Persister{}
}

func clone(orig []byte) []byte {
	x := make([]byte, len(orig))
	copy(x, orig)
	return x
}

func (ps *Persister) SaveRaftState(state []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	wd, err := os.Getwd()
	if err != nil {
		log.Print("cannot get working directory")
	}
	ofile, _ := ioutil.TempFile(wd, "temp"+raftStateFileName)
	ofile.Write(clone(state))
	os.Rename(ofile.Name(), raftStateFileName)
	ofile.Close()
}

func (ps *Persister) ReadRaftState() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	raftState, err := ReadFromDisk(raftStateFileName)
	if err != nil {
		return raftState
	} else {
		return nil
	}
}

func (ps *Persister) RaftStateSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	raftState, err := ReadFromDisk(raftStateFileName)
	if err != nil {
		return len(raftState)
	} else {
		return -1
	}
}

func ReadFromDisk(name string) ([]byte, error) {
	byteSlice := make([]byte, 0)
	err := error(nil)
	file, openErr := os.Open(name)
	if openErr != nil {
		log.Print(openErr)
		err = openErr
		return byteSlice, err
	}

	fileInfo, infoErr := file.Stat()
	if infoErr != nil {
		log.Print(infoErr)
		return byteSlice, err
	}

	byteSlice = make([]byte, fileInfo.Size())

	_, readErr := file.Read(byteSlice)
	if readErr != nil {
		log.Print(readErr)
		return byteSlice, err
	}

	file.Close()
	return byteSlice, err
}
