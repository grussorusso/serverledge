package utils

import (
	"net"
)

func GetIpAddress() (ipv4 net.IP) {
	tt, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, t := range tt {
		aa, err := t.Addrs()
		if err != nil {
			panic(err)
		}
		for _, a := range aa {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			v4 := ipnet.IP.To4()
			if v4 == nil || v4[0] == 127 { // do not consider loopBack address
				continue
			}
			return v4
		}
	}

	return nil
}
