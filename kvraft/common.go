package kvraft

type Err int

const (
	OK Err = iota
	ErrNoKey
	ErrWrongLeader
	ErrLockHeld
	ErrLockNotHeld
	ErrTimeout
)

type OpCode int

const (
	GetOp OpCode = iota
	PutOp
	AppendOp
	AcquireOp
	ReleaseOp
	RenewOp
	FailingAcquireOp
	FailingLockedOp
)

type RequestArgs struct {
	ClientId uint
	Seq      uint
	Code     OpCode
	Key      string
	Value    string
}

type RequestReply struct {
	Seq   uint
	E     Err
	Value string
}

const lockLeaseTime = 30
