package balloc

import (
	"log"
	"pp2/jrnl"
)

const startData = bitmapBlock + 1

func AllocBlock(t *jrnl.TxnHandle) uint {
	btmp := getBitmap()
	for i, bit := range btmp {
		if bit == 0 {
			setBit(btmp, uint(i))
			updateAndRelseBitmap(t, btmp)
			return uint(i) + startData
		}
	}

	log.Fatal("no blocks to alloc big sad")
	return 0
}

func RelseBlock(t *jrnl.TxnHandle, bn uint) {
	if bn < startData {
		log.Fatal("illegal block to relse")
	}
	bn = bn - startData
	btmp := getBitmap()
	if btmp[bn] == 0 {
		log.Fatal("double free in bitmap")
	}
	clearBit(btmp, bn)
	updateAndRelseBitmap(t, btmp)
}
