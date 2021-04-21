package netdrv

import (
	"errors"
	"fmt"
)

// In lieu of dynamically updatable configuration,
// use these lists to set up ncs
var kvList []string = []string{"192.168.0.19:1235", "192.168.0.20:1235", "192.168.0.18:1235"}
var rfList []string = []string{"192.168.0.19:1234", "192.168.0.20:1234", "192.168.0.18:1234"}

// Update this prior to compilation to
// set your own IP address
var myIp = "192.168.0.19"

func (c *NetConfig) GetMe() (int, error) {
	me := myIp + ":"
	if c.IsRaft {
		me += fmt.Sprintf("%d", c.RaftPort)
	} else {
		me += fmt.Sprintf("%d", c.RaftPort)
	}

	for i, srv := range c.Servers {
		if srv == me {
			return i, nil
		}
	}
	return 0, errors.New("i'm not in any server lists; check staticaddr.go")
}
