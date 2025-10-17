package collector

import (
	"bufio"
	"netfetch/internal/model"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func (c *Collector) collectOS() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	hostname, _ := os.Hostname()
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}

	osInfo := parseOSRelease()
	if osInfo == nil {
		osInfo = parseLSBRelease()
	}
	if osInfo == nil {
		osInfo = make(map[string]string)
	}

	c.info.OS = &model.OSInfo{
		Name:       getOrDefault(osInfo, "NAME", "Unknown"),
		PrettyName: getOrDefault(osInfo, "PRETTY_NAME", "Unknown"),
		Distro:     getOrDefault(osInfo, "ID", "Unknown"),
		IDLike:     getOrDefault(osInfo, "ID_LIKE", ""),
		Version:    getOrDefault(osInfo, "VERSION", ""),
		VersionID:  getOrDefault(osInfo, "VERSION_ID", ""),
		Codename:   getOrDefault(osInfo, "VERSION_CODENAME", ""),
		BuildID:    getOrDefault(osInfo, "BUILD_ID", ""),
		Variant:    getOrDefault(osInfo, "VARIANT", ""),
		VariantID:  getOrDefault(osInfo, "VARIANT_ID", ""),
		Arch:       getArchitecture(),
	}

	c.info.Host = hostname
	c.info.User = user
	c.info.Kernel = getKernelVersion()
}

func parseOSRelease() map[string]string {
	paths := []string{"/etc/os-release", "/usr/lib/os-release"}

	for _, path := range paths {
		if info := parseKeyValueFile(path); info != nil {
			return info
		}
	}
	return nil
}

func parseLSBRelease() map[string]string {
	info := parseKeyValueFile("/etc/lsb-release")
	if info == nil {
		return nil
	}

	normalized := make(map[string]string)
	normalized["NAME"] = getOrDefault(info, "DISTRIB_ID", "")
	normalized["VERSION"] = getOrDefault(info, "DISTRIB_RELEASE", "")
	normalized["ID"] = strings.ToLower(getOrDefault(info, "DISTRIB_ID", ""))
	normalized["VERSION_CODENAME"] = getOrDefault(info, "DISTRIB_CODENAME", "")
	normalized["PRETTY_NAME"] = getOrDefault(info, "DISTRIB_DESCRIPTION", "")

	return normalized
}

func parseKeyValueFile(path string) map[string]string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		value = strings.Trim(value, `"'`)

		result[key] = value
	}

	return result
}

func getKernelVersion() string {
	switch runtime.GOOS {
	case "linux", "darwin":
		out, err := exec.Command("uname", "-r").Output()
		if err != nil {
			return "Unknown"
		}
		return strings.TrimSpace(string(out))
	case "windows":
		return getWindowsVersion()
	default:
		return "Unknown"
	}
}

func getWindowsVersion() string {
	out, err := exec.Command("cmd", "/c", "ver").Output()
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(out))
}

func getArchitecture() string {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		return "x86_64"
	case "386":
		return "i686"
	case "arm64":
		return "aarch64"
	case "arm":
		return "armv7l"
	default:
		return arch
	}
}

func getUserShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = os.Getenv("COMSPEC")
	}
	if shell == "" {
		shell = "Unknown"
	}
	return shell
}

func getOrDefault(m map[string]string, key, defaultValue string) string {
	if val, ok := m[key]; ok && val != "" {
		return val
	}
	return defaultValue
}
