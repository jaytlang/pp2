package kvraft

type Err int

const (
	OK Err = iota
	ErrNoKey
	ErrWrongLeader
	ErrTimeout
)

type OpCode int

const (
	GetOp OpCode = iota
	PutOp
	AppendOp
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
