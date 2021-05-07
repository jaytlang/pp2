package balloc

import (
	"log"
	"pp2/jrnl"
)

var startData uint

func InitBalloc(dataStart uint) {
	startData = dataStart
}

// Might fail if messing with the bitmap
// also fails, but this is unlikely since we
// just grabbed it.
// CANNOT be invoked more than once per run,
// since changes don't hit the bitmap until txns
// complete!
func AllocBlocks(t *jrnl.TxnHandle, cnt uint) ([]uint, error) {
	btmp := getBitmap()
	blks := []uint{}

	for ; cnt > 0; cnt-- {
		for i, bit := range btmp {
			if bit == 0 {
				setBit(btmp, uint(i))
				if err := updateAndRelseBitmap(t, btmp); err != nil {
					return []uint{}, err
				}
				res := uint(i) + startData
				blks = append(blks, res)
			}
		}
	}

	if uint(len(blks)) < cnt {
		log.Fatal("no blocks to alloc big sad")
	}
	return blks, nil
}

// Might fail when messing with the bitmap fails.
// Note that changes don't hit the bitmap until txns
// complete, so can ONLY BE CALLED ONCE
func RelseBlocks(t *jrnl.TxnHandle, bns []uint) error {
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
	return updateAndRelseBitmap(t, btmp)
}
