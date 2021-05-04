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

const nServers = 3

func MkDefaultNetConfig(isRaft bool, register bool, nsAddr string) *NetConfig {
	c := NetConfig{
		KVPort:   defaultKVPort,
		RaftPort: defaultRFPort,
		IsRaft:   isRaft,
		Servers:  []string{},
	}

	// Register myself
	var me int
	if register {
		me = registerName(nsAddr)
	}
	for i := 0; i < nServers; i++ {
		if i == me && register {
			c.Servers = append(c.Servers, getMyIp())
		} else {
			c.Servers = append(c.Servers, getName(nsAddr, i))
		}
	}

	if isRaft {
		for idx, addr := range c.Servers {
			c.Servers[idx] = addr + ":" + fmt.Sprint(defaultRFPort)
		}
	} else {
		for idx, addr := range c.Servers {
			c.Servers[idx] = addr + ":" + fmt.Sprint(defaultKVPort)
		}
	}
	// fmt.Printf("CSERVERS: %v\n", c.Servers)
	return &c
}
