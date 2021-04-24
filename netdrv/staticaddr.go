package netdrv

import (
	"errors"
)

// In lieu of dynamically updatable configuration,
// use these lists to set up ncs
var kvList []string = []string{"192.168.0.19", "192.168.0.20", "192.168.0.10"}
var rfList []string = []string{"192.168.0.19", "192.168.0.20", "192.168.0.10"}

// Update this prior to compilation to
// set your own IP address
var myIp = "192.168.0.19"

func (c *NetConfig) GetMe() (int, error) {
	me := myIp
	for i, srv := range c.Servers {
		if srv == me {
			return i, nil
		}
	}
	return 0, errors.New("i'm not in any server lists; check staticaddr.go")
}
