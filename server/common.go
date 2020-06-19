package server

import "net"

//获取本地ip_v4地址
func getIpV4Address() string {
	adders, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, address := range adders {
		// 检查ip地址判断是否回环地址
		if inet, ok := address.(*net.IPNet); ok && !inet.IP.IsLoopback() {
			if inet.IP.To4() != nil {
				return inet.IP.String()
			}
		}
	}
	return ""
}
