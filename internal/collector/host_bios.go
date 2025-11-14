package collector

import (
	"netfetch/internal/model"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func (c *Collector) collectHostInfo() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		c.info.HostInfo = getHostInfoLinux()
	case "darwin":
		c.info.HostInfo = getHostInfoDarwin()
	case "windows":
		c.info.HostInfo = getHostInfoWindows()
	default:
		c.info.HostInfo = &model.HostInfo{}
	}
}

func (c *Collector) collectBIOS() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		c.info.BIOS = getBIOSLinux()
	case "darwin":
		c.info.BIOS = getBIOSDarwin()
	case "windows":
		c.info.BIOS = getBIOSWindows()
	default:
		c.info.BIOS = &model.BIOSInfo{}
	}
}

func getHostInfoLinux() *model.HostInfo {
	hostInfo := &model.HostInfo{}

	dmiBase := "/sys/class/dmi/id"

	if vendor, err := os.ReadFile(filepath.Join(dmiBase, "sys_vendor")); err == nil {
		hostInfo.Vendor = strings.TrimSpace(string(vendor))
	}

	if productName, err := os.ReadFile(filepath.Join(dmiBase, "product_name")); err == nil {
		hostInfo.Model = strings.TrimSpace(string(productName))
	}

	if productVersion, err := os.ReadFile(filepath.Join(dmiBase, "product_version")); err == nil {
		hostInfo.Version = strings.TrimSpace(string(productVersion))
	}

	if chassisType, err := os.ReadFile(filepath.Join(dmiBase, "chassis_type")); err == nil {
		typeNum := strings.TrimSpace(string(chassisType))
		hostInfo.Type = getChassisType(typeNum)
	}

	return hostInfo
}

func getBIOSLinux() *model.BIOSInfo {
	biosInfo := &model.BIOSInfo{}

	dmiBase := "/sys/class/dmi/id"

	if vendor, err := os.ReadFile(filepath.Join(dmiBase, "bios_vendor")); err == nil {
		biosInfo.Vendor = strings.TrimSpace(string(vendor))
	}

	if version, err := os.ReadFile(filepath.Join(dmiBase, "bios_version")); err == nil {
		biosInfo.Version = strings.TrimSpace(string(version))
	}

	if date, err := os.ReadFile(filepath.Join(dmiBase, "bios_date")); err == nil {
		biosInfo.Date = strings.TrimSpace(string(date))
	}

	if _, err := os.Stat("/sys/firmware/efi"); err == nil {
		biosInfo.Type = "UEFI"
	} else {
		biosInfo.Type = "Legacy"
	}

	return biosInfo
}

func getHostInfoDarwin() *model.HostInfo {
	hostInfo := &model.HostInfo{}

	if out, err := getSysctlStringErr("hw.model"); err == nil {
		hostInfo.Model = out
	}

	hostInfo.Vendor = "Apple Inc."
	hostInfo.Type = "Portable"

	return hostInfo
}

func getBIOSDarwin() *model.BIOSInfo {
	biosInfo := &model.BIOSInfo{}
	biosInfo.Type = "UEFI"
	biosInfo.Vendor = "Apple Inc."

	return biosInfo
}

func getHostInfoWindows() *model.HostInfo {
	hostInfo := &model.HostInfo{}

	if out, err := getWMICValue("computersystem", "Manufacturer"); err == nil {
		hostInfo.Vendor = out
	}

	if out, err := getWMICValue("computersystem", "Model"); err == nil {
		hostInfo.Model = out
	}

	return hostInfo
}

func getBIOSWindows() *model.BIOSInfo {
	biosInfo := &model.BIOSInfo{}

	if out, err := getWMICValue("bios", "Manufacturer"); err == nil {
		biosInfo.Vendor = out
	}

	if out, err := getWMICValue("bios", "SMBIOSBIOSVersion"); err == nil {
		biosInfo.Version = out
	}

	if out, err := getWMICValue("bios", "ReleaseDate"); err == nil {
		biosInfo.Date = out
	}

	biosInfo.Type = "UEFI"

	return biosInfo
}

func getChassisType(typeNum string) string {
	chassisTypes := map[string]string{
		"1":  "Other",
		"2":  "Unknown",
		"3":  "Desktop",
		"4":  "Low Profile Desktop",
		"5":  "Pizza Box",
		"6":  "Mini Tower",
		"7":  "Tower",
		"8":  "Portable",
		"9":  "Laptop",
		"10": "Notebook",
		"11": "Hand Held",
		"12": "Docking Station",
		"13": "All in One",
		"14": "Sub Notebook",
		"15": "Space-saving",
		"16": "Lunch Box",
		"17": "Main Server Chassis",
		"18": "Expansion Chassis",
		"19": "SubChassis",
		"20": "Bus Expansion Chassis",
		"21": "Peripheral Chassis",
		"22": "RAID Chassis",
		"23": "Rack Mount Chassis",
		"24": "Sealed-case PC",
		"25": "Multi-system",
		"26": "Compact PCI",
		"27": "Advanced TCA",
		"28": "Blade",
		"29": "Blade Enclosure",
		"30": "Tablet",
		"31": "Convertible",
		"32": "Detachable",
	}

	if chassisType, ok := chassisTypes[typeNum]; ok {
		return chassisType
	}
	return "Unknown"
}
