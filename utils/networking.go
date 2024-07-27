package utils

import (
	"net"
)

func GetIpAddress() (ipv4 net.IP) {
	return GetOutboundIP()
	//
	//tt, err := net.Interfaces()
	//if err != nil {
	//	panic(err)
	//}
	//for _, t := range tt {
	//	aa, err := t.Addrs()
	//	if err != nil {
	//		panic(err)
	//	}
	//	for _, a := range aa {
	//		ipnet, ok := a.(*net.IPNet)
	//		if !ok {
	//			continue
	//		}
	//		v4 := ipnet.IP.To4()
	//		if v4 == nil || v4[0] == 127 { // do not consider loopBack address
	//			continue
	//		}
	//		return v4
	//	}
	//}
	//
	//return nil
}

func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		panic(err)
	}

	defer func() {
		err1 := conn.Close()
		if err1 != nil {
			panic(err1)
		}
	}()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
