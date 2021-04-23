package kvraft

type Err int

const (
	OK Err = iota
	ErrNoKey
	ErrWrongLeader
	ErrLockHeld
	ErrTimeout
)

type OpCode int

const (
	GetOp OpCode = iota
	PutOp
	AppendOp
	AcquireOp
	FailingAcquireOp
	ReleaseOp
	FailingReleaseOp
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
