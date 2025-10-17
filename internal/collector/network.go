package collector

import (
	"net"
	"netfetch/internal/model"
)

func (c *Collector) collectNetwork() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	interfaces, err := net.Interfaces()
	if err != nil {
		c.info.Network.Interfaces = make([]model.InterfaceInfo, 0)
		return c.info.Network
	}

	c.info.Network.Interfaces = make([]model.InterfaceInfo, 0)

	for _, iface := range interfaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			if ipnet.IP.IsLoopback() {
				continue
			}

			if ipnet.IP.To4() != nil {
				c.info.Network.Interfaces = append(c.info.Network.Interfaces, model.InterfaceInfo{
					Name: iface.Name,
					IP:   ipnet.IP.String(),
				})
			}
		}
	}

	return c.info.Network
}

func (c *Collector) collectLocalIP() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var ips []string
	interfaces, err := net.Interfaces()
	if err != nil {
		c.info.LocalIP = ips
		return
	}

	for _, iface := range interfaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			if ipnet.IP.IsLoopback() {
				continue
			}

			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}

	c.info.LocalIP = ips
}
