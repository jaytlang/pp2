package jrnl

import (
	"fmt"
	"pp2/bio"
	"strconv"
	"strings"
)

func parseSb(blk *bio.Block) *logSB {
	lst := strings.Split(blk.Data, "/")
	if len(lst) != 3 {
		return &logSB{}
	}

	cnt, _ := strconv.ParseUint(lst[1], 10, 64)
	cmt, _ := strconv.ParseUint(lst[2], 10, 64)

	return &logSB{
		bitmap: lst[0],
		cnt:    uint(cnt),
		commit: uint(cmt),
	}
}

func flattenSb(sb *logSB) *bio.Block {
	data := fmt.Sprintf("%s/%d/%d", sb.bitmap, sb.cnt, sb.commit)
	return &bio.Block{
		Nr:   sbNr,
		Data: data,
	}
}

func parseLb(blk *bio.Block) *logBlock {
	lst := strings.Split(blk.Data, "/")
	if len(lst) < 3 {
		return &logBlock{lnr: blk.Nr}
	}

	rnr, _ := strconv.ParseUint(lst[0], 10, 64)

	var last bool
	if lst[2] == "1" {
		last = true
	}

	return &logBlock{
		lnr:   blk.Nr,
		rnr:   uint(rnr),
		rdata: lst[1],
		last:  last,
	}
}

func flattenLb(lb *logBlock) *bio.Block {
	ls := "0"
	if lb.last {
		ls = "1"
	}
	data := fmt.Sprintf("%d/%s/%s", lb.rnr, lb.rdata, ls)
	return &bio.Block{
		Nr:   lb.lnr,
		Data: data,
	}
}
