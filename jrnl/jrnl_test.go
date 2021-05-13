package jrnl

import (
	"pp2/bio"
	"testing"
)

// Tests the journal interface: Begin, Write, End, Abort
// Uses the mock disk + actual block layer

// Partitions:
//	-> Begin
//		-> No other, some other transactions running
//	-> End
//		-> No other, some other, many other transactions running
//		-> Committing, not committing
//	-> Abort
//		-> No other, some other transactions running
//		-> Some other transaction has to commit, not
//	-> WriteBlock
//		-> blk
//			-> Same block number is written twice in a txn
//			-> Same block number is written twice across txns
//		-> t
//			-> No other, some other transactions running
//			-> Transaction length == 1, >1 (and >> 1)

func initUut() {
	bio.Binit("", true)
	InitSb()
}

// Covers:
// 	- begin/txns/none
//	- end/txns/none
// 	- end/txns/committing
//	- write/t/nooverlappingtxns
//  - write/t/shorttxnlen
func TestSimple(tt *testing.T) {
	initUut()
	t := BeginTransaction()
	if err := t.WriteBlock(&bio.Block{
		Nr:   0,
		Data: "hello world",
	}); err != nil {
		tt.Errorf("failed to write block")
	}
	t.EndTransaction(false, true)

	b := bio.Bget(0)
	expect := bio.Block{
		Nr:   0,
		Data: "hello world",
	}

	if *b != expect {
		tt.Errorf("incorrect block: got %v/expected %v", *b, expect)
	}

	b.Brelse()
}

// Covers:
// 	- write/blk/acrosstxns
func TestConsecutive(tt *testing.T) {
	initUut()
	t1 := BeginTransaction()
	if err_t1 := t1.WriteBlock(&bio.Block{
		Nr:   0,
		Data: "firstTxn",
	}); err_t1 != nil {
		tt.Errorf("failed to write to block")
	}

	t1.EndTransaction(false, true)

	b := bio.Bget(0)
	expect := bio.Block{
		Nr:   0,
		Data: "firstTxn",
	}
	if *b != expect {
		tt.Errorf("incorrect block: got %v/expected %v", *b, expect)
	}
	b.Brelse()

	t2 := BeginTransaction()
	if err_t2 := t2.WriteBlock(&bio.Block{
		Nr:   0,
		Data: "secondTxn",
	}); err_t2 != nil {
		tt.Errorf("failed to write to block")
	}
	t2.EndTransaction(false, true)

	b = bio.Bget(0)
	expect.Data = "secondTxn"

	if *b != expect {
		tt.Errorf("incorrect block: got %v/expected %v", *b, expect)
	}
	b.Brelse()
}

// Covers:
//	- begin/txns/some
//	- end/txns/some
// 	- end/txns/notcommitting
//	- write/t/someoverlappingtxns
func TestNested(tt *testing.T) {
	initUut()
	t1 := BeginTransaction()
	t2 := BeginTransaction()

	if err := t1.WriteBlock(&bio.Block{
		Nr:   0,
		Data: "t1 write",
	}); err != nil {
		tt.Errorf("failed to write to block")
	}

	if err := t2.WriteBlock(&bio.Block{
		Nr:   1,
		Data: "t2 write",
	}); err != nil {
		tt.Errorf("failed to write to block")
	}

	t2.EndTransaction(false, false)
	t1.EndTransaction(false, false)

	b := bio.Bget(0)
	expect := bio.Block{
		Nr:   0,
		Data: "t1 write",
	}
	if *b != expect {
		tt.Errorf("incorrect block: got %v/expected %v", *b, expect)
	}
	b.Brelse()

	b = bio.Bget(1)
	expect = bio.Block{
		Nr:   1,
		Data: "t2 write",
	}
	if *b != expect {
		tt.Errorf("incorrect block: got %v/expected %v", *b, expect)
	}
	b.Brelse()
}

// Covers:
//	- writeblock/b/blockwrittentwicepertxn
func TestOverlapping(tt *testing.T) {
	initUut()
	t1 := BeginTransaction()
	t2 := BeginTransaction()

	if err := t1.WriteBlock(&bio.Block{
		Nr:   0,
		Data: "t1 write",
	}); err != nil {
		tt.Errorf("failed to write to block")
	}
	t1.EndTransaction(false, false)

	if err := t2.WriteBlock(&bio.Block{
		Nr:   1,
		Data: "WRONG",
	}); err != nil {
		tt.Errorf("failed to write to block")
	}
	if err := t2.WriteBlock(&bio.Block{
		Nr:   1,
		Data: "t2 write",
	}); err != nil {
		tt.Errorf("failed to write to block")
	}
	t2.EndTransaction(false, false)

	b := bio.Bget(0)
	expect := bio.Block{
		Nr:   0,
		Data: "t1 write",
	}
	if *b != expect {
		tt.Errorf("incorrect block: got %v/expected %v", *b, expect)
	}
	b.Brelse()

	b = bio.Bget(1)
	expect = bio.Block{
		Nr:   1,
		Data: "t2 write",
	}
	if *b != expect {
		tt.Errorf("incorrect block: got %v/expected %v", *b, expect)
	}
	b.Brelse()
}

// Covers:
//	- write/t/largetxnlen
func TestManyWrites(tt *testing.T) {
	initUut()
	t := BeginTransaction()
	for i := uint(EndJrnl); i < EndJrnl+blkPerSys; i++ {
		if err := t.WriteBlock(&bio.Block{
			Nr:   i,
			Data: "i'm a transaction lol",
		}); err != nil {
			tt.Errorf("failed to write to block")
		}
	}
	t.EndTransaction(false, true)

	for i := uint(EndJrnl); i < EndJrnl+blkPerSys; i++ {
		b := bio.Bget(i)
		expect := bio.Block{
			Nr:   uint(i),
			Data: "i'm a transaction lol",
		}
		if *b != expect {
			tt.Errorf("mismatching block %d: got %v vs. %v\n", i, *b, expect)
			break
		}
		b.Brelse()
	}
}

// Covers:
//	- end/txns/many
func TestManyTransactions(tt *testing.T) {
	initUut()
	tArr := make([]*TxnHandle, 0)
	for i := uint(EndJrnl); i < EndJrnl+sysPerLog; i++ {
		tArr = append(tArr, BeginTransaction())
	}

	for j := uint(EndJrnl); j < EndJrnl+sysPerLog; j++ {
		if err := tArr[j-uint(EndJrnl)].WriteBlock(&bio.Block{
			Nr:   uint(j),
			Data: "i'm a transaction lol",
		}); err != nil {
			tt.Errorf("failed to write to block")
		}
	}

	for k := uint(EndJrnl); k < EndJrnl+sysPerLog; k++ {
		tArr[k-uint(EndJrnl)].EndTransaction(false, false)
	}

	for i := uint(EndJrnl); i < EndJrnl+sysPerLog; i++ {
		b := bio.Bget(i)
		expect := bio.Block{
			Nr:   uint(i),
			Data: "i'm a transaction lol",
		}
		if *b != expect {
			tt.Errorf("mismatching block %d: got %v vs. %v\n", i, *b, expect)
			break
		}
		b.Brelse()
	}
}

// Covers:
//	- abort/txns/none
//	- abort/nocommits
func TestSimpleAbort(tt *testing.T) {
	initUut()
	t := BeginTransaction()

	if err := t.WriteBlock(&bio.Block{
		Nr:   0,
		Data: "i'm a transaction lol",
	}); err != nil {
		tt.Errorf("failed to write to block")
	}

	t.AbortTransaction(true)

	b := bio.Bget(0)
	expect := bio.Block{
		Nr:   0,
		Data: "",
	}
	if *b != expect {
		tt.Errorf("mismatching block: got %v vs. %v\n", *b, expect)
	}
}

func TestMultipleAbort(tt *testing.T) {
	initUut()
	t1 := BeginTransaction()
	t2 := BeginTransaction()

	if err := t1.WriteBlock(&bio.Block{
		Nr:   0,
		Data: "t1 write",
	}); err != nil {
		tt.Errorf("failed to write to block")
	}
	t1.EndTransaction(false, false)

	if err := t2.WriteBlock(&bio.Block{
		Nr:   1,
		Data: "bad",
	}); err != nil {
		tt.Errorf("failed to write to block")
	}
	t2.AbortTransaction(true)

	b := bio.Bget(0)
	expect := bio.Block{
		Nr:   0,
		Data: "t1 write",
	}
	if *b != expect {
		tt.Errorf("incorrect block: got %v/expected %v", *b, expect)
	}
	b.Brelse()

	b = bio.Bget(1)
	expect = bio.Block{
		Nr:   1,
		Data: "",
	}
	if *b != expect {
		tt.Errorf("incorrect block: got %v/expected %v", *b, expect)
	}
	b.Brelse()

}
