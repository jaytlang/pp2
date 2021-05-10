package balloc

import (
	"fmt"
	"log"
	"pp2/jrnl"
)

var startData uint

func InitBalloc(dataStart uint) {
	startData = dataStart
}

// CANNOT be invoked more than once per run,
// since changes don't hit the bitmap until txns
// complete!
// Always succeeds.
func AllocBlocks(t *jrnl.TxnHandle, cnt uint) []uint {
retry:
	btmp := getBitmap()
	blks := []uint{}
	fmt.Printf("Trying to alloc %d blocks\n", cnt)

	for ; cnt > 0; cnt-- {
		for i, bit := range btmp {
			if bit == 0 {
				setBit(btmp, uint(i))
				res := uint(i) + startData
				blks = append(blks, res)
				break
			}
		}
	}

	// Checkme
	if uint(len(blks)) < cnt {
		log.Fatal("no blocks to alloc big sad")
	}

	if err := updateAndRelseBitmap(t, btmp); err != nil {
		// We lost the bitmap. Try again...
		goto retry
	}
	fmt.Printf("allocated %d blocks\n", len(blks))
	return blks
}

// CAN ONLY BE CALLED ONCE
// Will always succeed.
func RelseBlocks(t *jrnl.TxnHandle, bns []uint) {
retry:
	btmp := getBitmap()
	for _, bn := range bns {
		if bn < startData {
			log.Fatal("illegal block to relse")
		}
		bn = bn - startData
		if btmp[bn] == 0 {
			log.Fatal("double free in bitmap")
		}
		clearBit(btmp, bn)
	}
	if err := updateAndRelseBitmap(t, btmp); err != nil {
		goto retry
	}
}
