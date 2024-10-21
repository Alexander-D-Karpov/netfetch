package collector

import (
	"net"
	"netfetch/internal/model"
)

func (c *Collector) collectNetwork() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	interfaces, _ := net.Interfaces()
	c.info.Network.Interfaces = make([]model.InterfaceInfo, 0)

	for _, iface := range interfaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue // interface down or loopback interface
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					c.info.Network.Interfaces = append(c.info.Network.Interfaces, model.InterfaceInfo{
						Name: iface.Name,
						IP:   ipnet.IP.String(),
					})
				}
			}
		}
	}

	return c.info.Network
}
