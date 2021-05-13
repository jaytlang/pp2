package balloc

import (
	"pp2/bio"
	"pp2/jrnl"
	"testing"
)

// Partitions
//	-> Alloc
//		-> 1 block, more than 1 block
//		-> 1 alloc, many allocs
//		-> previously released blocks alloced
//	-> Release
//		-> 1 block, more than 1 block
//		-> 1 alloc, many allocs

func initUut() {
	bio.Binit("", true)
	jrnl.InitSb()
	InitBalloc(jrnl.EndJrnl + 1)
}

// Covers:
//	-> alloc/1blk
//	-> alloc/1alloc
//	-> release/1blk
//	-> release/1alloc
func TestSimpleAlloc(tt *testing.T) {
	initUut()
	t := jrnl.BeginTransaction()
	blks := AllocBlocks(t, 1)
	if len(blks) != 1 {
		tt.Errorf("1 block not allocated: got %v\n", blks)
	}

	t.EndTransaction(false, true)
	RelseBlocks(t, blks)
}

// Covers:
//	-> alloc/>1blk
//	-> release/>1blk
func TestMultiBlockAlloc(tt *testing.T) {
	initUut()
	t := jrnl.BeginTransaction()
	blks := AllocBlocks(t, 5)
	if len(blks) != 5 {
		tt.Errorf("5 blocks not allocated: got %d, wanted 5\n", len(blks))
	}

	t.EndTransaction(false, true)
	RelseBlocks(t, blks)
}

// Covers:
//	-> alloc/>1alloc
//	-> release/>1alloc
func TestMoreThanOneAlloc(tt *testing.T) {
	initUut()
	t := jrnl.BeginTransaction()
	blk1 := AllocBlocks(t, 1)
	t.EndTransaction(false, true)

	if len(blk1) != 1 {
		tt.Errorf("1 block not allocated: got %v\n", blk1)
	}

	t = jrnl.BeginTransaction()
	blk2 := AllocBlocks(t, 1)
	t.EndTransaction(false, true)

	if len(blk2) != 1 {
		tt.Errorf("1 block not allocated: got %v\n", blk2)
	}

	if blk1[0] == blk2[0] {
		tt.Errorf("Block numbers should be distinct but both are: %v", blk1[0])
	}
}

// Covers:
//	-> alloc/prevblocks
func TestBigAlloc(tt *testing.T) {
	initUut()
	t := jrnl.BeginTransaction()
	b := AllocBlocks(t, 4096)
	t.EndTransaction(false, true)

	if len(b) != 4096 {
		tt.Errorf("4096 blocks not allocated: got %v\n", b)
	}

	t = jrnl.BeginTransaction()
	RelseBlocks(t, b)
	t.EndTransaction(false, true)

	t = jrnl.BeginTransaction()
	b = AllocBlocks(t, 4096)
	t.EndTransaction(false, true)

	if len(b) != 4096 {
		tt.Errorf("4096 blocks not allocated: got %v\n", b)
	}
}
