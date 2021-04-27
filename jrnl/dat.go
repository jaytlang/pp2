package jrnl

// The journal generally takes advantage
// of the fact that we aren't dealing with
// hard block sizes, rather we have a KV
// server that can support pretty much arbitrary
// length strings

// Also, note that we use a lot of string
// types bc we're big lazy. We also take advantage
// of implicit locking from the block layer

type logBlock struct {
	lnr   uint
	rnr   uint
	rdata string
	last  bool
}

type logSB struct {
	bitmap string
	cnt    uint
	commit uint
}

const sbNr = 2
const logStart = 3
const blkPerSys = 100
const sysPerLog = 1000
