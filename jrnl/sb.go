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
