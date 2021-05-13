package fs

import (
	"errors"
	"fmt"
	"pp2/inode"
	"pp2/jrnl"
	"strconv"
	"strings"
)

type Filesystem struct {
	rooti   uint16
	fdTable map[int]*File // file desc -> inode num
	maxFd   int
}

type File struct {
	inum   uint16
	offset uint
}

func (f *Filesystem) mkFd() int {
	f.maxFd++
	return f.maxFd - 1
}

func Mount() *Filesystem {
	f := new(Filesystem)
	f.fdTable = make(map[int]*File)

	if !inode.Probei(0) {
		t := jrnl.BeginTransaction()
		root := inode.Alloci(t, inode.Dir)
		root.Relse()
		t.EndTransaction(false, true)
	}

	f.rooti = 0
	return f
}

func (f *Filesystem) Open(fname string) int {
	// Read the root directory
	// Of the form "filename,inum filename,inum filename,inum"
	var inum uint16

	rawroot := inode.Readi(f.rooti, 0, ^uint(0))
	entries := strings.Split(rawroot, " ")
	found := false
	for _, entry := range entries {
		data := strings.Split(entry, ",")
		if data[0] == fname {
			found = true
			fmt.Printf("Found file %s\n", fname)

			inum64, _ := strconv.ParseUint(data[1], 10, 16)
			inum = uint16(inum64)
			break
		}
	}

	if !found {
		// Create the file
		fmt.Printf("File %s not found, making it\n", fname)

		t := jrnl.BeginTransaction()
		newi := inode.Alloci(t, inode.File)
		inode.Writei(t, f.rooti, uint(len(rawroot)), fmt.Sprintf("%s,%d ", fname, newi.Serialnum))
		inum = newi.Serialnum

		t.EndTransaction(false, true)
		newi.Relse()
		fmt.Printf("Made new file %s\n", fname)
	}

	newFd := f.mkFd()
	f.fdTable[newFd] = &File{
		inum: inum,
	}

	return newFd
}

func (f *Filesystem) Read(fd int, count uint) (string, error) {
	if _, ok := f.fdTable[fd]; !ok {
		return "", errors.New("no such fd")
	}

	file := f.fdTable[fd]
	content := inode.Readi(file.inum, file.offset, count)
	file.offset += uint(len(content))
	return content, nil
}

func (f *Filesystem) Write(fd int, data string) (uint, error) {
	if _, ok := f.fdTable[fd]; !ok {
		return 0, errors.New("no such fd")
	}

	file := f.fdTable[fd]

	t := jrnl.BeginTransaction()
	cnt, err := inode.Writei(t, file.inum, file.offset, data)
	if err != nil {
		t.AbortTransaction(true)
		return 0, err
	}

	t.EndTransaction(false, true)
	file.offset += cnt
	return cnt, nil

}

func (f *Filesystem) Close(fd int) {
	delete(f.fdTable, fd)
}
