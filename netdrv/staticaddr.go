package netdrv

import (
	"errors"
	"fmt"
)

// In lieu of dynamically updatable configuration,
// use these lists to set up ncs
var ipList []string = []string{"10.0.0.197", "10.0.0.186", "10.0.0.128"}

// Update this prior to compilation to
// set your own IP address
var myIp = "10.0.0.197"

func (c *NetConfig) GetMe() (int, error) {
	me := myIp
	if c.IsRaft {
		me += ":1234"
	} else {
		me += ":1235"
	}

	fmt.Printf("%v in %v?\n", me, c.Servers)
	for i, srv := range c.Servers {
		if srv == me {
			return i, nil
		}
	}
	return 0, errors.New("i'm not in any server lists; check staticaddr.go")
}
