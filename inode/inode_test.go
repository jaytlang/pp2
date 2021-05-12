package inode

import (
	"fmt"
	"pp2/balloc"
	"pp2/bio"
	"pp2/jrnl"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Tests the inode api:
//	-> Readi, Writei, Alloci, Freei, Relsei,
//	-> Geti, EnqWrite

// Partitions:
//	-> Readi
//		-> offset = 0; offset <= len(file); > end
//		-> count = 0; count <= len(file); count > len(file)
//	-> Writei
//		-> offset = 0; <= len(file); > end (=FAIL)
//		-> len(data) = 0; > 0; >maxValid (=FAIL)
//	-> Alloci
//		-> 1 alloc, many allocs
//		-> previously released blocks alloced
//	-> Freei
//		-> 1 alloc, many allocs

func initUut() {
	bio.Binit("", true)
	jrnl.InitSb()
	balloc.InitBalloc(EndInode)
	InodeInit()
}

// Covers:
//	-> readi/offset/0
//	-> readi/count/0
//	-> writei/offset/0
//	-> writei/lendata/0
//	-> alloci/1alloc
//	-> freei/1alloc
func TestEmptyReadWrite(tt *testing.T) {
	initUut()
	t := jrnl.BeginTransaction()
	i := Alloci(t, File)
	expect := Inode{
		Serialnum: 0,
		Refcnt:    1,
		Filesize:  0,
		Addrs:     []uint{},
		Mode:      File,
	}

	if !cmp.Equal(expect, *i) {
		tt.Errorf("didn't get right inode, got %v/wanted %v\n", *i, expect)
	}
	t.EndTransaction(false)

	t = jrnl.BeginTransaction()
	cnt, err := Writei(t, 0, 0, "")
	if cnt != 0 {
		tt.Errorf("didn't write zero bytes, wrote %d\n", cnt)
	} else if err != nil {
		tt.Errorf("error during write")
	}

	t.EndTransaction(false)
	data := Readi(0, 0, 0)
	if data != "" {
		tt.Errorf("didn't read empty data as expected")
	}

	t = jrnl.BeginTransaction()
	if i.Free(t) != nil {
		tt.Errorf("error freeing inode")
	}
	t.EndTransaction(false)
}

// Covers:
//	-> readi/count/>len
//	-> writei/lendata/>0
func TestBasicReadWrite(tt *testing.T) {
	initUut()
	t := jrnl.BeginTransaction()
	i := Alloci(t, File)
	expect := Inode{
		Serialnum: 0,
		Refcnt:    1,
		Filesize:  0,
		Addrs:     []uint{},
		Mode:      File,
	}

	if !cmp.Equal(expect, *i) {
		tt.Errorf("didn't get right inode, got %v/wanted %v\n", *i, expect)
	}
	t.EndTransaction(false)

	t = jrnl.BeginTransaction()
	cnt, err := Writei(t, 0, 0, strings.Repeat("hello", 4097))

	if cnt != 4097*5 {
		tt.Errorf("didn't write 4097*5 bytes, wrote %d\n", cnt)
	} else if err != nil {
		tt.Errorf("error during write")
	}

	t.EndTransaction(false)

	data := Readi(0, 0, 99999)
	if data != strings.Repeat("hello", 4097) {
		tt.Errorf("didn't read 4097*5 bytes as expected")
	}

	t = jrnl.BeginTransaction()
	if i.Free(t) != nil {
		tt.Errorf("error freeing inode")
	}
	t.EndTransaction(false)
}

// Covers:
//	-> write/lendata/thicc
func TestMassiveWrite(tt *testing.T) {
	initUut()
	t := jrnl.BeginTransaction()
	i := Alloci(t, File)
	expect := Inode{
		Serialnum: 0,
		Refcnt:    1,
		Filesize:  0,
		Addrs:     []uint{},
		Mode:      File,
	}

	if !cmp.Equal(expect, *i) {
		tt.Errorf("didn't get right inode, got %v/wanted %v\n", *i, expect)
	}
	t.EndTransaction(false)

	t = jrnl.BeginTransaction()
	s := strings.Repeat("h", 3000000)
	_, err := Writei(t, 0, 0, s)

	if err == nil {
		tt.Errorf("seriously dude?")
	}

	t.AbortTransaction()

	t = jrnl.BeginTransaction()
	if i.Free(t) != nil {
		tt.Errorf("error freeing inode")
	}
	t.EndTransaction(false)
}

// Covers:
//	-> alloci/manyallocs
//	-> freei/manyallocs
func TestDoubleAlloc(tt *testing.T) {
	initUut()
	t := jrnl.BeginTransaction()
	i1 := Alloci(t, File)
	t.EndTransaction(false)

	t = jrnl.BeginTransaction()
	i2 := Alloci(t, File)
	t.EndTransaction(false)

	if cmp.Equal(i1, i2) {
		tt.Errorf("allocated same inode twice")
	}
	t = jrnl.BeginTransaction()
	i1.Free(t)
	i2.Free(t)
	t.EndTransaction(false)
}

// Covers:
//	-> alloci/prevreleased
// Oh shit
func TestStressAlloc(tt *testing.T) {
	initUut()

	for iter := 0; iter < 1; iter++ {
		txns := make([]*jrnl.TxnHandle, 0)
		allocs := make([]*Inode, 0)

		fmt.Printf("\t****Transacting...\n")
		for i := 0; i < 100; i++ {
			txns = append(txns, jrnl.BeginTransaction())
			fmt.Printf("Getting %d\n", i)
			allocs = append(allocs, Alloci(txns[i], File))
			txns[i].EndTransaction(false)
		}

		fmt.Printf("\t****Freeing...\n")
		t := jrnl.BeginTransaction()
		for m := 0; m < 100; m++ {
			allocs[m].Free(t)
		}
		t.EndTransaction(false)
	}

}

// Covers:
//	-> readi/offset/<len
//	-> readi/count/<len
// 	-> writei/offset/<len
func TestSmallOffset(tt *testing.T) {
	initUut()
	t := jrnl.BeginTransaction()
	i1 := Alloci(t, File)
	t.EndTransaction(false)

	t = jrnl.BeginTransaction()
	cnt, err := Writei(t, i1.Serialnum, 0, strings.Repeat("a", 10))
	if cnt != 10 {
		tt.Errorf("couldn't write 10 bytes")
	} else if err != nil {
		tt.Errorf("error during initial write")
	}
	t.EndTransaction(false)

	t = jrnl.BeginTransaction()
	cnt, err = Writei(t, i1.Serialnum, 3, "bbb")
	if cnt != 3 {
		tt.Errorf("didn't write 3 bytes")
	} else if err != nil {
		tt.Errorf("error during follow-on offsetted write")
	}
	t.EndTransaction(false)

	dat := Readi(i1.Serialnum, 2, 7)
	expect := "abbb"
	if dat != expect {
		tt.Errorf("read %v vs. expected %v\n", dat, expect)
	}

	t = jrnl.BeginTransaction()
	i1.Free(t)
	t.EndTransaction(false)
}

// Covers:
//	-> readi/offset/>end
//	-> writei/offset/>end
func TestBigOffsets(tt *testing.T) {
	initUut()
	t := jrnl.BeginTransaction()
	i1 := Alloci(t, File)
	t.EndTransaction(false)

	t = jrnl.BeginTransaction()
	cnt, err := Writei(t, i1.Serialnum, 0, strings.Repeat("a", 10))
	if cnt != 10 {
		tt.Errorf("couldn't write 10 bytes")
	} else if err != nil {
		tt.Errorf("error during initial write")
	}
	t.EndTransaction(false)

	data := Readi(i1.Serialnum, 50, 10)
	if data != "" {
		tt.Errorf("somehow read stuff off the end")
	}

	t = jrnl.BeginTransaction()
	_, err = Writei(t, i1.Serialnum, 500, "what")
	if err == nil {
		tt.Errorf("somehow wrote off the end")
	}

	t.AbortTransaction()
	t = jrnl.BeginTransaction()
	i1.Free(t)
	t.EndTransaction(false)
}
