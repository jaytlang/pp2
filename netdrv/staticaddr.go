package netdrv

import (
	"errors"
	"fmt"
)

// In lieu of dynamically updatable configuration,
// use these lists to set up ncs
var ipList []string = []string{"192.168.0.19", "192.168.0.20", "192.168.0.10"}

// Update this prior to compilation to
// set your own IP address
var myIp = "192.168.0.19"

func (c *NetConfig) GetMe() (int, error) {
	me := myIp
	if c.IsRaft {
		me += ":1234"
	} else {
		me += ":1235"
	}

	for i, srv := range c.Servers {
		if srv == me {
			return i, nil
		}
	}
	return 0, errors.New("i'm not in any server lists; check staticaddr.go")
}
