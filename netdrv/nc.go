package netdrv

import (
	"log"
	"net/rpc"
)

// Servers includes ports in address strings
// these have to be consistent with the KVPort/RaftPort
type NetConfig struct {
	IsRaft   bool
	Servers  []string
	KVPort   uint16
	RaftPort uint16
}

const defaultKVPort = 1235
const defaultRFPort = 1234

func (c *NetConfig) DialAll() []*rpc.Client {
	l := []*rpc.Client{}
	for _, addr := range c.Servers {
		c, err := rpc.DialHTTP("tcp", addr)
		if err != nil {
			log.Fatal("failed to dial peer:", err)
		}
		l = append(l, c)
	}
	return l
}

func MkDefaultNetConfig(isRaft bool) *NetConfig {
	c := &NetConfig{
		KVPort:   defaultKVPort,
		RaftPort: defaultRFPort,
	}

	if isRaft {
		c.Servers = rfList
	} else {
		c.Servers = kvList
	}
	return c
}
