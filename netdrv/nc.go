package netdrv

import (
	"fmt"
	"net/rpc"
	"time"
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
		for {
			c, err := rpc.DialHTTP("tcp", addr)
			if err != nil {
				fmt.Printf("Failed to dial %s, retrying...\n", addr)
				time.Sleep(time.Second)
				continue
			}
			fmt.Printf("Dialed %s\n", addr)
			l = append(l, c)
			break
		}
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
		for idx, addr := range c.Servers {
			c.Servers[idx] = addr + fmt.Sprint(defaultRFPort)
		}
	} else {
		c.Servers = kvList
		for idx, addr := range c.Servers {
			c.Servers[idx] = addr + fmt.Sprint(defaultKVPort)
		}
	}
	return c
}
