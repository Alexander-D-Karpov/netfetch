package collector

import (
	"netfetch/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func (c *Collector) collectBattery() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		collectBatteryLinux(c.info)
	case "darwin":
		collectBatteryDarwin(c.info)
	case "windows":
		collectBatteryWindows(c.info)
	case "freebsd":
		collectBatteryBSD(c.info)
	}
}

func (c *Collector) collectPowerAdapter() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		collectPowerAdapterLinux(c.info)
	case "darwin":
		collectPowerAdapterDarwin(c.info)
	case "windows":
		collectPowerAdapterWindows(c.info)
	}
}

func collectBatteryLinux(info *model.SystemInfo) {
	batteryDirs, err := filepath.Glob("/sys/class/power_supply/BAT*")
	if err != nil || len(batteryDirs) == 0 {
		batteryDirs, err = filepath.Glob("/sys/class/power_supply/battery")
		if err != nil || len(batteryDirs) == 0 {
			return
		}
	}

	for _, batteryPath := range batteryDirs {
		capacityData, err := os.ReadFile(filepath.Join(batteryPath, "capacity"))
		if err != nil {
			continue
		}

		statusData, err := os.ReadFile(filepath.Join(batteryPath, "status"))
		if err != nil {
			continue
		}

		capacityStr := strings.TrimSpace(string(capacityData))
		capacity, err := strconv.ParseFloat(capacityStr, 64)
		if err != nil {
			continue
		}

		status := strings.TrimSpace(string(statusData))

		batteryInfo := &model.BatteryInfo{
			Percentage: capacity,
			Status:     "",
		}

		if status == "Charging" {
			batteryInfo.Status = "Charging"
		} else if status == "Discharging" {
			batteryInfo.Status = "Discharging"
		} else if status == "Full" {
			batteryInfo.Status = "Full"
		} else if status == "Not charging" {
			batteryInfo.Status = "Not Charging"
		} else {
			batteryInfo.Status = status
		}

		acOnline := false
		adapterDirs, err := filepath.Glob("/sys/class/power_supply/AC*")
		if err == nil && len(adapterDirs) > 0 {
			onlineData, err := os.ReadFile(filepath.Join(adapterDirs[0], "online"))
			if err == nil {
				onlineStr := strings.TrimSpace(string(onlineData))
				acOnline = onlineStr == "1"
			}
		}

		if acOnline && batteryInfo.Status != "Full" {
			if batteryInfo.Status != "" {
				batteryInfo.Status += ", AC Connected"
			} else {
				batteryInfo.Status = "AC Connected"
			}
		} else if acOnline && batteryInfo.Status == "Full" {
			batteryInfo.Status = "Full, AC Connected"
		}

		info.Battery = batteryInfo
		return
	}
}

func collectPowerAdapterLinux(info *model.SystemInfo) {
	adapterDirs, err := filepath.Glob("/sys/class/power_supply/AC*")
	if err != nil || len(adapterDirs) == 0 {
		adapterDirs, err = filepath.Glob("/sys/class/power_supply/ADP*")
		if err != nil || len(adapterDirs) == 0 {
			return
		}
	}

	adapterPath := adapterDirs[0]

	onlineData, err := os.ReadFile(filepath.Join(adapterPath, "online"))
	if err != nil {
		return
	}

	onlineStr := strings.TrimSpace(string(onlineData))
	isConnected := onlineStr == "1"

	info.PowerAdapter = &model.PowerAdapterInfo{
		IsConnected: isConnected,
	}
}

func collectBatteryDarwin(info *model.SystemInfo) {
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return
	}

	batteryLine := lines[1]

	var percentage float64
	var status string

	if strings.Contains(batteryLine, "%") {
		parts := strings.Fields(batteryLine)
		for i, part := range parts {
			if strings.HasSuffix(part, "%") || strings.HasSuffix(part, "%;") {
				percentStr := strings.TrimSuffix(strings.TrimSuffix(part, "%"), ";")
				if p, err := strconv.ParseFloat(percentStr, 64); err == nil {
					percentage = p
				}

				if i+1 < len(parts) {
					rawStatus := strings.Trim(parts[i+1], ";")
					if rawStatus == "charging" {
						status = "Charging, AC Connected"
					} else if rawStatus == "discharging" {
						status = "Discharging"
					} else if rawStatus == "charged" {
						status = "Full, AC Connected"
					} else {
						status = rawStatus
					}
				}
				break
			}
		}
	}

	if percentage > 0 {
		info.Battery = &model.BatteryInfo{
			Percentage: percentage,
			Status:     status,
		}
	}
}

func collectPowerAdapterDarwin(info *model.SystemInfo) {
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return
	}

	output := string(out)
	isConnected := strings.Contains(output, "AC Power")

	info.PowerAdapter = &model.PowerAdapterInfo{
		IsConnected: isConnected,
	}
}

func collectBatteryWindows(info *model.SystemInfo) {
	out, err := exec.Command("wmic", "path", "Win32_Battery", "get", "EstimatedChargeRemaining,BatteryStatus", "/format:list").Output()
	if err != nil {
		return
	}

	var percentage float64
	var batteryStatus int

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "EstimatedChargeRemaining=") {
			percentStr := strings.TrimPrefix(line, "EstimatedChargeRemaining=")
			if p, err := strconv.ParseFloat(percentStr, 64); err == nil {
				percentage = p
			}
		} else if strings.HasPrefix(line, "BatteryStatus=") {
			statusStr := strings.TrimPrefix(line, "BatteryStatus=")
			if s, err := strconv.Atoi(statusStr); err == nil {
				batteryStatus = s
			}
		}
	}

	status := ""
	switch batteryStatus {
	case 1:
		status = "Discharging"
	case 2:
		status = "Charging, AC Connected"
	case 3:
		status = "Full, AC Connected"
	default:
		status = "Unknown"
	}

	if percentage > 0 {
		info.Battery = &model.BatteryInfo{
			Percentage: percentage,
			Status:     status,
		}
	}
}

func collectPowerAdapterWindows(info *model.SystemInfo) {
	out, err := exec.Command("wmic", "path", "Win32_Battery", "get", "BatteryStatus", "/format:list").Output()
	if err != nil {
		return
	}

	isConnected := false
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "BatteryStatus=") {
			statusStr := strings.TrimPrefix(line, "BatteryStatus=")
			if status, err := strconv.Atoi(statusStr); err == nil {
				isConnected = status == 2 || status == 3
			}
		}
	}

	info.PowerAdapter = &model.PowerAdapterInfo{
		IsConnected: isConnected,
	}
}

func collectBatteryBSD(info *model.SystemInfo) {
	out, err := exec.Command("sysctl", "-n", "hw.acpi.battery.life").Output()
	if err != nil {
		return
	}

	percentStr := strings.TrimSpace(string(out))
	percentage, err := strconv.ParseFloat(percentStr, 64)
	if err != nil {
		return
	}

	statusOut, _ := exec.Command("sysctl", "-n", "hw.acpi.battery.state").Output()
	statusStr := strings.TrimSpace(string(statusOut))

	status := "Unknown"
	switch statusStr {
	case "0":
		status = "Full"
	case "1":
		status = "Discharging"
	case "2":
		status = "Charging"
	}

	acOut, _ := exec.Command("sysctl", "-n", "hw.acpi.acline").Output()
	acStr := strings.TrimSpace(string(acOut))
	if acStr == "1" {
		if status != "Full" {
			status += ", AC Connected"
		}
	}

	info.Battery = &model.BatteryInfo{
		Percentage: percentage,
		Status:     status,
	}
}
