package jrnl

import (
	"fmt"
	"pp2/bio"
)

func InitSb() {
	sb := parseSb(bio.Bget(sbNr))
	var nsb *bio.Block

	if sb.commit > 0 {
		resurrect(sb)
		goto done
	}

	sb.bitmap = ""
	for i := 0; i < sysPerLog; i++ {
		sb.bitmap += "0"
	}
	sb.cnt = 0
	sb.commit = 0

	nsb = flattenSb(sb)
	nsb.Bpush()

done:
	nsb.Brelse()
	fmt.Printf("Superblock initialized successfully\n")
}

// Start a transaction. Updates internal
// metadata to ensure consistency and
// returns the syscall log subset in which
// this person is to write, NOT the raw
// block number
func beginTransaction() uint {
	var res uint
start:
	sb := parseSb(bio.Bget(sbNr))
	for i, c := range sb.bitmap {
		if c == '0' {
			ob := []rune(sb.bitmap)
			ob[i] = '1'
			sb.bitmap = string(ob)

			res = uint(i)
			goto done
		}
	}

	// Retry if no luck, i.e. log is outta room
	flattenSb(sb).Brelse()
	goto start

done:
	sb.cnt++
	nsb := flattenSb(sb)
	nsb.Bpush()
	nsb.Brelse()

	fmt.Printf("Began transaction in log segment %d\n", res)
	return res
}

func endTransaction() {
	sb := parseSb(bio.Bget(sbNr))
	sb.cnt--
	fmt.Printf("Finished a transaction\n")

	// TODO
	if sb.cnt == 0 {
		fmt.Printf("Outstanding transactions to zero, committing...")
		commit(sb)
	}
	flattenSb(sb).Brelse()
}
