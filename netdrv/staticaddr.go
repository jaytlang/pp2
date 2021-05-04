package netdrv

import (
	"errors"
)

func (c *NetConfig) GetMe() (int, error) {
	me := getMyIp()
	if c.IsRaft {
		me += ":1234"
	} else {
		me += ":1235"
	}

	//fmt.Printf("%v in %v?\n", me, c.Servers)
	for i, srv := range c.Servers {
		if srv == me {
			return i, nil
		}
	}
	return 0, errors.New("i'm not in any server lists; check staticaddr.go")
}
