package collector

import (
	"io"
	"net"
	"net/http"
	"netfetch/internal/model"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
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

func (c *Collector) collectPublicIP() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.info.PublicIP = getPublicIP()
}

func (c *Collector) collectWifi() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		c.info.Wifi = getWifiLinux()
	case "darwin":
		c.info.Wifi = getWifiDarwin()
	case "windows":
		c.info.Wifi = getWifiWindows()
	default:
		c.info.Wifi = nil
	}
}

func getPublicIP() string {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me",
		"https://icanhazip.com",
	}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ip := strings.TrimSpace(string(body))
		if ip != "" {
			return ip
		}
	}

	return ""
}

func getWifiLinux() *model.WifiInfo {
	out, err := exec.Command("nmcli", "-t", "-f", "active,ssid,chan,rate,signal,security", "dev", "wifi").Output()
	if err != nil {
		return getWifiLinuxIw()
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) < 6 {
			continue
		}

		if fields[0] == "yes" {
			wifi := &model.WifiInfo{
				SSID:     fields[1],
				Strength: 0,
			}

			if channel := strings.TrimSpace(fields[2]); channel != "" {
				if ch, err := strconv.Atoi(channel); err == nil {
					if ch >= 1 && ch <= 14 {
						wifi.Frequency = "2.4 GHz"
					} else if ch >= 36 {
						wifi.Frequency = "5 GHz"
					}
				}
			}

			if rate := strings.TrimSpace(fields[3]); rate != "" {
				wifi.Protocol = parseWifiProtocol(rate)
			}

			if signal := strings.TrimSpace(fields[4]); signal != "" {
				if sig, err := strconv.Atoi(signal); err == nil {
					wifi.Strength = sig
				}
			}

			if security := strings.TrimSpace(fields[5]); security != "" {
				wifi.Security = security
			} else {
				wifi.Security = "Open"
			}

			return wifi
		}
	}

	return nil
}

func getWifiLinuxIw() *model.WifiInfo {
	out, err := exec.Command("iw", "dev").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(out), "\n")
	var interfaceName string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Interface ") {
			interfaceName = strings.TrimPrefix(line, "Interface ")
		}
	}

	if interfaceName == "" {
		return nil
	}

	linkOut, err := exec.Command("iw", "dev", interfaceName, "link").Output()
	if err != nil {
		return nil
	}

	wifi := &model.WifiInfo{}
	linkLines := strings.Split(string(linkOut), "\n")

	for _, line := range linkLines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "SSID:") {
			wifi.SSID = strings.TrimSpace(strings.TrimPrefix(line, "SSID:"))
		} else if strings.HasPrefix(line, "freq:") {
			freqStr := strings.TrimSpace(strings.TrimPrefix(line, "freq:"))
			if freq, err := strconv.Atoi(freqStr); err == nil {
				if freq < 3000 {
					wifi.Frequency = "2.4 GHz"
				} else {
					wifi.Frequency = "5 GHz"
				}
			}
		} else if strings.Contains(line, "signal:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "signal:" && i+2 < len(parts) {
					signalStr := strings.TrimSpace(parts[i+1])
					if sig, err := strconv.Atoi(signalStr); err == nil {
						wifi.Strength = (sig + 110) * 10 / 7
						if wifi.Strength > 100 {
							wifi.Strength = 100
						}
					}
				}
			}
		}
	}

	if wifi.SSID == "" {
		return nil
	}

	return wifi
}

func getWifiDarwin() *model.WifiInfo {
	out, err := exec.Command("/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport", "-I").Output()
	if err != nil {
		return nil
	}

	wifi := &model.WifiInfo{}
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "SSID:") {
			wifi.SSID = strings.TrimSpace(strings.TrimPrefix(line, "SSID:"))
		} else if strings.HasPrefix(line, "channel:") {
			channelStr := strings.TrimSpace(strings.TrimPrefix(line, "channel:"))
			if ch, err := strconv.Atoi(strings.Split(channelStr, ",")[0]); err == nil {
				if ch >= 1 && ch <= 14 {
					wifi.Frequency = "2.4 GHz"
				} else {
					wifi.Frequency = "5 GHz"
				}
			}
		} else if strings.HasPrefix(line, "lastTxRate:") {
			rateStr := strings.TrimSpace(strings.TrimPrefix(line, "lastTxRate:"))
			wifi.Protocol = parseWifiProtocol(rateStr)
		} else if strings.HasPrefix(line, "agrCtlRSSI:") {
			rssiStr := strings.TrimSpace(strings.TrimPrefix(line, "agrCtlRSSI:"))
			if rssi, err := strconv.Atoi(rssiStr); err == nil {
				wifi.Strength = (rssi + 110) * 10 / 7
				if wifi.Strength > 100 {
					wifi.Strength = 100
				}
			}
		} else if strings.HasPrefix(line, "link auth:") {
			authStr := strings.TrimSpace(strings.TrimPrefix(line, "link auth:"))
			wifi.Security = authStr
		}
	}

	if wifi.SSID == "" {
		return nil
	}

	return wifi
}

func getWifiWindows() *model.WifiInfo {
	out, err := exec.Command("netsh", "wlan", "show", "interfaces").Output()
	if err != nil {
		return nil
	}

	wifi := &model.WifiInfo{}
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "SSID") && !strings.Contains(line, "BSSID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				wifi.SSID = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Radio type") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				radioType := strings.TrimSpace(parts[1])
				if strings.Contains(radioType, "802.11n") {
					wifi.Protocol = "Wi-Fi 4 (802.11n)"
				} else if strings.Contains(radioType, "802.11ac") {
					wifi.Protocol = "Wi-Fi 5 (802.11ac)"
				} else if strings.Contains(radioType, "802.11ax") {
					wifi.Protocol = "Wi-Fi 6 (802.11ax)"
				}
			}
		} else if strings.Contains(line, "Channel") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				channelStr := strings.TrimSpace(parts[1])
				if ch, err := strconv.Atoi(channelStr); err == nil {
					if ch >= 1 && ch <= 14 {
						wifi.Frequency = "2.4 GHz"
					} else {
						wifi.Frequency = "5 GHz"
					}
				}
			}
		} else if strings.Contains(line, "Signal") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				signalStr := strings.TrimSuffix(strings.TrimSpace(parts[1]), "%")
				if sig, err := strconv.Atoi(signalStr); err == nil {
					wifi.Strength = sig
				}
			}
		} else if strings.Contains(line, "Authentication") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				wifi.Security = strings.TrimSpace(parts[1])
			}
		}
	}

	if wifi.SSID == "" {
		return nil
	}

	return wifi
}

func parseWifiProtocol(rateOrProtocol string) string {
	rateStr := strings.ToLower(rateOrProtocol)

	if strings.Contains(rateStr, "ax") || strings.Contains(rateStr, "wifi 6") {
		return "Wi-Fi 6 (802.11ax)"
	} else if strings.Contains(rateStr, "ac") || strings.Contains(rateStr, "wifi 5") {
		return "Wi-Fi 5 (802.11ac)"
	} else if strings.Contains(rateStr, "n") || strings.Contains(rateStr, "wifi 4") {
		return "Wi-Fi 4 (802.11n)"
	} else if strings.Contains(rateStr, "g") {
		return "802.11g"
	} else if strings.Contains(rateStr, "a") {
		return "802.11a"
	} else if strings.Contains(rateStr, "b") {
		return "802.11b"
	}

	if rate, err := strconv.Atoi(strings.Fields(rateStr)[0]); err == nil {
		if rate >= 1000 {
			return "Wi-Fi 6 (802.11ax)"
		} else if rate >= 400 {
			return "Wi-Fi 5 (802.11ac)"
		} else if rate >= 100 {
			return "Wi-Fi 4 (802.11n)"
		} else if rate >= 54 {
			return "802.11g"
		} else if rate >= 11 {
			return "802.11b"
		}
	}

	return "Unknown"
}
