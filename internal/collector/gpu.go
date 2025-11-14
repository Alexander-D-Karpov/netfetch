package collector

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func (c *Collector) collectGPU() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		c.info.GPU = getGPULinux()
	case "darwin":
		c.info.GPU = getGPUDarwin()
	case "windows":
		c.info.GPU = getGPUWindows()
	case "freebsd", "openbsd", "netbsd":
		c.info.GPU = getGPUBSD()
	default:
		c.info.GPU = "Unknown"
	}
}

func getGPULinux() string {
	gpus := detectGPUDRM()

	if len(gpus) == 0 {
		gpus = detectGPULspci()
	}

	if len(gpus) == 0 {
		return "Unknown"
	}

	return strings.Join(gpus, ", ")
}

func detectGPUDRM() []string {
	var gpus []string

	cardDirs, err := filepath.Glob("/sys/class/drm/card[0-9]*/device")
	if err != nil {
		return gpus
	}

	seen := make(map[string]bool)

	for _, cardDir := range cardDirs {
		modalias, err := os.ReadFile(filepath.Join(cardDir, "modalias"))
		if err != nil {
			continue
		}

		modaliasStr := strings.TrimSpace(string(modalias))
		if !strings.HasPrefix(modaliasStr, "pci:") {
			continue
		}

		var vendorID, deviceID, classID string
		re := regexp.MustCompile(`v([0-9A-Fa-f]{8})d([0-9A-Fa-f]{8}).*bc([0-9A-Fa-f]{2})`)
		matches := re.FindStringSubmatch(modaliasStr)

		if len(matches) >= 4 {
			vendorID = matches[1][4:]
			deviceID = matches[2][4:]
			classID = matches[3]

			if classID != "03" {
				continue
			}

			gpuName := lookupGPUName(vendorID, deviceID, cardDir)

			key := vendorID + ":" + deviceID
			if !seen[key] {
				seen[key] = true
				if gpuName != "" {
					gpus = append(gpus, gpuName)
				}
			}
		}
	}

	return gpus
}

func lookupGPUName(vendorID, deviceID, sysPath string) string {
	driverName := ""
	driverLink, err := os.Readlink(filepath.Join(sysPath, "driver"))
	if err == nil {
		driverName = filepath.Base(driverLink)
	}

	var name string

	switch driverName {
	case "amdgpu", "radeon":
		name = getAMDGPUName(sysPath)
	case "nvidia", "nouveau":
		name = getNVIDIAGPUName(sysPath)
	case "i915", "xe":
		name = getIntelGPUName(sysPath)
	}

	if name == "" {
		name = readPCIDatabase(vendorID, deviceID)
	}

	if name == "" {
		name = fmt.Sprintf("GPU [%s:%s]", vendorID, deviceID)
	}

	return name
}

func getAMDGPUName(sysPath string) string {
	if data, err := os.ReadFile(filepath.Join(sysPath, "product_name")); err == nil {
		name := strings.TrimSpace(string(data))
		if name != "" {
			return name
		}
	}

	return ""
}

func getNVIDIAGPUName(sysPath string) string {
	if data, err := os.ReadFile(filepath.Join(sysPath, "label")); err == nil {
		name := strings.TrimSpace(string(data))
		if name != "" {
			return name
		}
	}

	return ""
}

func getIntelGPUName(sysPath string) string {
	if data, err := os.ReadFile(filepath.Join(sysPath, "device")); err == nil {
		deviceID := strings.TrimSpace(string(data))
		deviceID = strings.TrimPrefix(deviceID, "0x")
		return readPCIDatabase("8086", deviceID)
	}

	return ""
}

func readPCIDatabase(vendorID, deviceID string) string {
	pciIDPaths := []string{
		"/usr/share/hwdata/pci.ids",
		"/usr/share/misc/pci.ids",
		"/var/lib/pciutils/pci.ids",
		"/usr/local/share/pciids/pci.ids",
	}

	for _, path := range pciIDPaths {
		if name := searchPCIDatabase(path, vendorID, deviceID); name != "" {
			return name
		}
	}

	return ""
}

func searchPCIDatabase(path, vendorID, deviceID string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	vendorID = strings.ToLower(vendorID)
	deviceID = strings.ToLower(deviceID)

	inVendor := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#") || len(line) == 0 {
			continue
		}

		if !strings.HasPrefix(line, "\t") {
			if strings.HasPrefix(strings.ToLower(line), vendorID) {
				inVendor = true
			} else {
				inVendor = false
			}
		} else if inVendor && strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "\t\t") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && strings.ToLower(parts[0]) == deviceID {
				return strings.Join(parts[1:], " ")
			}
		}
	}

	return ""
}

func detectGPULspci() []string {
	out, err := exec.Command("lspci").Output()
	if err != nil {
		return nil
	}

	var gpus []string
	re := regexp.MustCompile(`(VGA compatible controller|3D controller|Display controller): (.*)`)
	matches := re.FindAllStringSubmatch(string(out), -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 3 {
			gpu := strings.TrimSpace(match[2])
			gpu = cleanGPUName(gpu)
			if !seen[gpu] {
				seen[gpu] = true
				gpus = append(gpus, gpu)
			}
		}
	}

	return gpus
}

func cleanGPUName(name string) string {
	parts := strings.Split(name, ":")
	if len(parts) > 1 {
		name = strings.TrimSpace(parts[1])
	}

	vendors := map[string]string{
		"NVIDIA Corporation":           "NVIDIA",
		"Advanced Micro Devices, Inc.": "AMD",
		"Intel Corporation":            "Intel",
		"ATI Technologies Inc":         "ATI",
	}

	for oldVendor, newVendor := range vendors {
		if strings.HasPrefix(name, oldVendor) {
			name = strings.TrimPrefix(name, oldVendor)
			name = strings.TrimSpace(name)
			name = newVendor + " " + name
			break
		}
	}

	if strings.Contains(name, "[") && strings.Contains(name, "]") {
		start := strings.Index(name, "[")
		end := strings.Index(name, "]")
		if start < end {
			prefix := strings.TrimSpace(name[:start])
			bracketContent := strings.TrimSpace(name[start+1 : end])

			if prefix == "" {
				name = bracketContent
			} else {
				hasVendor := false
				for _, vendor := range vendors {
					if strings.Contains(prefix, vendor) || strings.Contains(bracketContent, vendor) {
						hasVendor = true
						break
					}
				}

				if hasVendor && strings.Contains(bracketContent, prefix) {
					name = bracketContent
				} else if hasVendor {
					name = prefix + " " + bracketContent
				} else {
					name = bracketContent
				}
			}
		}
	}

	name = regexp.MustCompile(`\(.*?\)`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	name = strings.TrimSpace(name)

	return name
}

func getGPUDarwin() string {
	out, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return "Unknown"
	}

	lines := strings.Split(string(out), "\n")
	var gpus []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Chipset Model:") {
			gpu := strings.TrimSpace(strings.TrimPrefix(line, "Chipset Model:"))
			gpus = append(gpus, gpu)
		}
	}

	if len(gpus) == 0 {
		return "Unknown"
	}

	return strings.Join(gpus, ", ")
}

func getGPUWindows() string {
	out, err := exec.Command("wmic", "path", "win32_VideoController", "get", "name", "/format:list").Output()
	if err != nil {
		return "Unknown"
	}

	var gpus []string
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name=") {
			gpu := strings.TrimPrefix(line, "Name=")
			if gpu != "" {
				gpus = append(gpus, gpu)
			}
		}
	}

	if len(gpus) == 0 {
		return "Unknown"
	}

	return strings.Join(gpus, ", ")
}

func getGPUBSD() string {
	out, err := exec.Command("pciconf", "-lv").Output()
	if err != nil {
		return "Unknown"
	}

	var gpus []string
	lines := strings.Split(string(out), "\n")

	inDisplay := false
	for _, line := range lines {
		if strings.Contains(line, "class=0x03") {
			inDisplay = true
		} else if inDisplay && strings.HasPrefix(strings.TrimSpace(line), "device") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				gpu := strings.Trim(parts[1], "' ")
				gpus = append(gpus, gpu)
			}
			inDisplay = false
		} else if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			inDisplay = false
		}
	}

	if len(gpus) == 0 {
		return "Unknown"
	}

	return strings.Join(gpus, ", ")
}
