package balloc

import (
	"pp2/bio"
	"pp2/inode"
	"pp2/jrnl"
)

const bitmapBlock = inode.EndInode + 1

type bitmap []byte

func getBitmap() bitmap {
	blk := bio.Bget(bitmapBlock)
	if blk.Data == "" {
		blk.Data = string(make([]byte, 4096))
	}
	return bitmap(blk.Data)
}

func setBit(b bitmap, nr uint) {
	b[nr] = 0x1
}

func clearBit(b bitmap, nr uint) {
	b[nr] = 0x0
}

func updateAndRelseBitmap(t *jrnl.TxnHandle, b bitmap) {
	blk := bio.Block{
		Nr:   bitmapBlock,
		Data: string(b),
	}

	t.WriteBlock(&blk)
	blk.Brelse()
}
