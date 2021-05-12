package bio

import (
	"testing"
)

// Test the bio interface, Binit/Bget/Bpush/Brelse/Brenew,
// in complete isolation assuming that the disk works.

// Partitions:
// Bget:
//	-> Nr
//		-> Corresponds to empty block, doesn't
//		-> Block lock is held (=HANG), isn't
//		-> Is very large, is very small, isn't either
// Bpush:
//	-> b
//		-> Data has been changed, hasn't
//		-> Data was empty, isn't now
//		-> Block number is very large, is very small, neither
// 		-> Block lock is held, isn't (=FAILURE)
// Brenew:
//	-> b
//		-> Block lock is held, isn't (=FAILURE)
// Brelse:
//	-> b
//		-> Block lock is held, isn't (=FAILURE)
//		-> Block data does not persist independent of Bpush

// Covers:
//	- bget/nr/emptyb
//  - bget/nr/nonemptyb
//  - bget/nr/small
//  - bpush/b/changed
//  - bpush/b/wasempty
//  - bpush/b/nrsmall
// 	- bpush/b/held
//  - brelse/b/held
func TestSetEmpty(t *testing.T) {
	Binit("", true)

	b := Bget(0)
	defer b.Brelse()

	expect := Block{
		Nr:   0,
		Data: "",
	}
	if *b != expect {
		t.Errorf("got %v instead of %v\n", *b, expect)
	}

	b.Data = "this is a test!"
	expect.Data = b.Data
	if b.Bpush() != OK {
		t.Errorf("got BioError pushing held block\n")
	}

	if b.Brelse() != OK {
		t.Errorf("got BioError releasing held block\n")
	}

	if *Bget(0) != expect {
		t.Errorf("got %v instead of %v after persistence", *b, expect)
	}
}

// Covers:
//	- bget/nr/large
//  - bpush/b/nrlarge
func TestSetLarge(t *testing.T) {
	Binit("", true)

	b := Bget(^uint(0))
	defer b.Brelse()

	expect := Block{
		Nr:   ^uint(0),
		Data: "",
	}
	if *b != expect {
		t.Errorf("got %v instead of %v\n", *b, expect)
	}

	b.Data = "this is a test again!"
	expect.Data = b.Data
	if b.Bpush() != OK {
		t.Errorf("got BioError pushing held block\n")
	}

	if b.Brelse() != OK {
		t.Errorf("got BioError releasing held block\n")
	}

	if *Bget(^uint(0)) != expect {
		t.Errorf("got %v instead of %v after persistence\n", *b, expect)
	}
}

// Covers:
//	- bget/nr/neither
//	- brenew/b/held
//  - brenew/b/unheld
func TestRenew(t *testing.T) {
	Binit("", true)
	b := Bget(5)
	expect := Block{
		Nr:   5,
		Data: "",
	}
	if *b != expect {
		t.Errorf("got %v instead of %v\n", *b, expect)
	}

	err := b.Brenew()
	if *b != expect {
		t.Errorf("got %v instead of %v\n", *b, expect)
	} else if err != OK {
		t.Errorf("failed to renew held block\n")
	}

	err = b.Brelse()
	if err != OK {
		t.Errorf("failed to release held block\n")
	}

	err = b.Brenew()
	if err == OK {
		t.Errorf("somehow renewed freed block\n")
	}
}

// Covers:
// 	- bpush/b/neither
//	- bpush/b/notheld
//  - brelse/b/notheld
func TestUnheldPushRelse(t *testing.T) {
	Binit("", true)
	b := Bget(5)
	expect := Block{
		Nr:   5,
		Data: "",
	}
	if *b != expect {
		t.Errorf("got %v instead of %v\n", *b, expect)
	}

	b.Data = "testing"
	expect.Data = b.Data
	if b.Bpush() != OK {
		t.Errorf("failed to push held block\n")
	}

	err := b.Brelse()
	if err != OK {
		t.Errorf("failed to release held block\n")
	}
	b = Bget(5)
	if *b != expect {
		t.Errorf("got %v instead of %v\n", *b, expect)
	}
	err = b.Brelse()
	if err != OK {
		t.Errorf("failed to release held block\n")
	}

	b.Data = "WRONG"
	if b.Bpush() == OK {
		t.Errorf("pushed block that's not held\n")
	}

	if b.Brelse() == OK {
		t.Errorf("released block that's not held\n")
	}

	b = Bget(5)
	if *b != expect {
		t.Errorf("got %v instead of %v\n", *b, expect)
	}
}

// Covers:
//	- bpush/b/unchanged
//	- brelse/b/persistence
func TestTransientUpdate(t *testing.T) {
	Binit("", true)
	b := Bget(5)
	expect := Block{
		Nr:   5,
		Data: "",
	}
	if *b != expect {
		t.Errorf("got %v instead of %v\n", *b, expect)
	}

	b.Data = "testing"
	expect.Data = b.Data
	if b.Bpush() != OK {
		t.Errorf("failed to push held block\n")
	}

	err := b.Brelse()
	if err != OK {
		t.Errorf("failed to release held block\n")
	}

	b = Bget(5)
	if *b != expect {
		t.Errorf("got %v instead of %v\n", *b, expect)
	}

	if b.Bpush() != OK {
		t.Errorf("failed to push held, unchanged block\n")
	}

	b.Data = "WRONG"
	if b.Brelse() != OK {
		t.Errorf("failed to release held block\n")
	}

	b = Bget(5)
	if *b != expect {
		t.Errorf("got %v instead of %v\n", *b, expect)
	}

}
