package collector

import (
	"bufio"
	"fmt"
	"netfetch/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func parseKeyValueFileWithScanner(path string) map[string]string {
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

func readFirstLine(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}

	return ""
}

func readAllLines(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func getEnvWithDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TiB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GiB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MiB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KiB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatPercentage(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return (float64(used) / float64(total)) * 100
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func unique(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

func (c *Collector) collectDateTime() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.info.DateTime = time.Now().Format("2006-01-02 15:04:05")
}

func (c *Collector) collectUsers() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux", "darwin":
		c.info.Users = getUsersUnix()
	case "windows":
		c.info.Users = getUsersWindows()
	default:
		c.info.Users = []model.UserInfo{}
	}
}

func (c *Collector) collectBrightness() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		c.info.Brightness = getBrightnessLinux()
	case "darwin":
		c.info.Brightness = getBrightnessDarwin()
	case "windows":
		c.info.Brightness = getBrightnessWindows()
	default:
		c.info.Brightness = nil
	}
}

func (c *Collector) collectLoginManager() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch runtime.GOOS {
	case "linux":
		c.info.LoginManager = getLoginManagerLinux()
	case "darwin":
		c.info.LoginManager = "macOS Login Window"
	case "windows":
		c.info.LoginManager = "Windows Login"
	default:
		c.info.LoginManager = "Unknown"
	}
}

func getUsersUnix() []model.UserInfo {
	out, err := exec.Command("who").Output()
	if err != nil {
		return []model.UserInfo{}
	}

	lines := strings.Split(string(out), "\n")
	var users []model.UserInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		user := model.UserInfo{
			Name:      fields[0],
			Terminal:  fields[1],
			LoginTime: strings.Join(fields[2:4], " "),
		}

		users = append(users, user)
	}

	return users
}

func getUsersWindows() []model.UserInfo {
	out, err := exec.Command("query", "user").Output()
	if err != nil {
		return []model.UserInfo{}
	}

	lines := strings.Split(string(out), "\n")
	var users []model.UserInfo

	for i, line := range lines {
		if i == 0 {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		user := model.UserInfo{
			Name:     fields[0],
			Terminal: fields[1],
		}

		if len(fields) >= 5 {
			user.LoginTime = fields[4]
		}

		users = append(users, user)
	}

	return users
}

func getBrightnessLinux() *model.BrightnessInfo {
	backlightDirs, err := filepath.Glob("/sys/class/backlight/*")
	if err != nil || len(backlightDirs) == 0 {
		return nil
	}

	backlightDir := backlightDirs[0]

	currentData, err := os.ReadFile(filepath.Join(backlightDir, "brightness"))
	if err != nil {
		return nil
	}

	maxData, err := os.ReadFile(filepath.Join(backlightDir, "max_brightness"))
	if err != nil {
		return nil
	}

	current, err := strconv.Atoi(strings.TrimSpace(string(currentData)))
	if err != nil {
		return nil
	}

	max, err := strconv.Atoi(strings.TrimSpace(string(maxData)))
	if err != nil {
		return nil
	}

	return &model.BrightnessInfo{
		Current: (current * 100) / max,
		Max:     100,
	}
}

func getBrightnessDarwin() *model.BrightnessInfo {
	out, err := exec.Command("brightness", "-l").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "brightness") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				brightnessStr := strings.TrimSpace(fields[1])
				if brightness, err := strconv.ParseFloat(brightnessStr, 64); err == nil {
					return &model.BrightnessInfo{
						Current: int(brightness * 100),
						Max:     100,
					}
				}
			}
		}
	}

	return nil
}

func getBrightnessWindows() *model.BrightnessInfo {
	out, err := exec.Command("powershell", "-Command", "(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightness).CurrentBrightness").Output()
	if err != nil {
		return nil
	}

	brightnessStr := strings.TrimSpace(string(out))
	if brightness, err := strconv.Atoi(brightnessStr); err == nil {
		return &model.BrightnessInfo{
			Current: brightness,
			Max:     100,
		}
	}

	return nil
}

func getLoginManagerLinux() string {
	displayManagerProcesses := map[string]string{
		"gdm":      "GDM",
		"gdm3":     "GDM3",
		"lightdm":  "LightDM",
		"sddm":     "SDDM",
		"lxdm":     "LXDM",
		"slim":     "SLiM",
		"xdm":      "XDM",
		"kdm":      "KDM",
		"mdm":      "MDM",
		"nodm":     "nodm",
		"entrance": "Entrance",
		"ly":       "Ly",
		"lemurs":   "Lemurs",
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return "Unknown"
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid := entry.Name()
		if pid[0] < '0' || pid[0] > '9' {
			continue
		}

		cmdlinePath := filepath.Join("/proc", pid, "cmdline")
		cmdline, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue
		}

		cmd := strings.Split(string(cmdline), "\x00")[0]
		exe := filepath.Base(cmd)

		if dm, ok := displayManagerProcesses[exe]; ok {
			return dm
		}
	}

	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "Wayland (Unknown DM)"
	}

	if os.Getenv("DISPLAY") != "" {
		return "X11 (Unknown DM)"
	}

	return "Unknown"
}

func getSysctlStringErr(key string) (string, error) {
	out, err := exec.Command("sysctl", "-n", key).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getWMICValue(class, property string) (string, error) {
	out, err := exec.Command("wmic", class, "get", property, "/format:list").Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(out), "\n")
	prefix := property + "="

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			value := strings.TrimPrefix(line, prefix)
			return strings.TrimSpace(value), nil
		}
	}

	return "", fmt.Errorf("property not found")
}
