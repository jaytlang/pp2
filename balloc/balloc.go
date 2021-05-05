package balloc

import (
	"log"
	"pp2/jrnl"
)

const startData = bitmapBlock + 1

func AllocBlock(t *jrnl.TxnHandle) (uint, error) {
	btmp := getBitmap()
	for i, bit := range btmp {
		if bit == 0 {
			setBit(btmp, uint(i))
			if err := updateAndRelseBitmap(t, btmp); err != nil {
				return 0, err
			}
			return uint(i) + startData, nil
		}
	}

	log.Fatal("no blocks to alloc big sad")
	// Never reached
	return 0, nil
}

func RelseBlock(t *jrnl.TxnHandle, bn uint) error {
	if bn < startData {
		log.Fatal("illegal block to relse")
	}
	bn = bn - startData
	btmp := getBitmap()
	if btmp[bn] == 0 {
		log.Fatal("double free in bitmap")
	}
	clearBit(btmp, bn)
	return updateAndRelseBitmap(t, btmp)
}
