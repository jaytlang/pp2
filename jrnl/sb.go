package jrnl

import (
	"fmt"
	"pp2/bio"
)

func InitSb() {
retry:
	sb := parseSb(bio.Bget(sbNr))
	nsb := flattenSb(sb)

	if sb.commit > 0 {
		err := resurrect(sb)
		if err != nil {
			goto retry
		}
		goto done
	}

	sb.bitmap = ""
	for i := 0; i < sysPerLog; i++ {
		sb.bitmap += "0"
	}
	sb.cnt = 0
	sb.commit = 0

	nsb = flattenSb(sb)
	if nsb.Bpush() != bio.OK {
		// Just goto retry...not
		// much else we can really do
		goto retry
	}

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
	if sb.commit > 0 {
		flattenSb(sb).Brelse()
		goto start
	}
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
	err := nsb.Bpush()
	if err != bio.OK {
		goto start
	}
	nsb.Brelse()

	fmt.Printf("Began transaction in log segment %d\n", res)
	return res
}

func endTransaction() {
retry:
	sb := parseSb(bio.Bget(sbNr))
	if sb.commit > 0 {
		flattenSb(sb).Brelse()
		goto retry
	}
	sb.cnt--
	fmt.Printf("Finished a transaction\n")

	if sb.cnt == 0 {
		fmt.Printf("Outstanding transactions to zero, committing...")
		err := commit(sb)
		if err != nil {
			goto retry
		}
	} else {
		err := flattenSb(sb).Bpush()
		if err != bio.OK {
			goto retry
		}
	}
	flattenSb(sb).Brelse()
}
