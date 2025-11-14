package collector

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type PackageCount struct {
	Manager string
	Count   int
}

func (c *Collector) collectPackages() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.activeModules["packages"] {
		c.info.Packages = "Unknown"
		return
	}

	managers := getPackageManagers()
	counts := make(chan PackageCount, len(managers))
	var wg sync.WaitGroup

	for _, manager := range managers {
		wg.Add(1)
		go func(m string) {
			defer wg.Done()
			count := countPackages(m)
			if count > 0 {
				counts <- PackageCount{Manager: m, Count: count}
			}
		}(manager)
	}

	go func() {
		wg.Wait()
		close(counts)
	}()

	totalPackages := 0
	var details []string

	for count := range counts {
		totalPackages += count.Count
		details = append(details, fmt.Sprintf("%d (%s)", count.Count, count.Manager))
	}

	if totalPackages > 0 {
		if len(details) == 1 {
			c.info.Packages = details[0]
		} else {
			c.info.Packages = strings.Join(details, ", ")
		}
	} else {
		c.info.Packages = "Unknown"
	}
}

func getPackageManagers() []string {
	managers := []string{}

	switch runtime.GOOS {
	case "linux":
		if pathExists("/var/lib/dpkg/status") {
			managers = append(managers, "dpkg")
		}
		if pathExists("/var/lib/pacman/local") {
			managers = append(managers, "pacman")
		}
		if pathExists("/var/lib/rpm") {
			managers = append(managers, "rpm")
		}
		if pathExists("/var/db/pkg") {
			managers = append(managers, "emerge")
		}
		if pathExists("/var/lib/flatpak/app") || pathExists(filepath.Join(os.Getenv("HOME"), ".local/share/flatpak/app")) {
			managers = append(managers, "flatpak")
		}
		if pathExists("/snap") {
			managers = append(managers, "snap")
		}
		if pathExists("/nix/var/nix/profiles") {
			managers = append(managers, "nix")
		}

	case "darwin":
		if pathExists("/usr/local/Cellar") || pathExists("/opt/homebrew/Cellar") {
			managers = append(managers, "brew")
		}
		if pathExists("/nix/var/nix/profiles") {
			managers = append(managers, "nix")
		}

	case "freebsd", "openbsd", "netbsd":
		if pathExists("/var/db/pkg") {
			managers = append(managers, "pkg")
		}
	}

	return managers
}

func countPackages(manager string) int {
	switch manager {
	case "dpkg":
		return countDpkg()
	case "pacman":
		return countPacman()
	case "rpm":
		return countRPM()
	case "emerge":
		return countEmerge()
	case "flatpak":
		return countFlatpak()
	case "snap":
		return countSnap()
	case "nix":
		return countNix()
	case "brew":
		return countBrew()
	case "pkg":
		return countPkgBSD()
	default:
		return 0
	}
}

func countDpkg() int {
	file, err := os.Open("/var/lib/dpkg/status")
	if err != nil {
		return 0
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	inPackage := false
	isInstalled := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Package:") {
			inPackage = true
			isInstalled = false
		} else if inPackage && strings.HasPrefix(line, "Status:") {
			if strings.Contains(line, "install ok installed") {
				isInstalled = true
			}
		} else if inPackage && line == "" {
			if isInstalled {
				count++
			}
			inPackage = false
			isInstalled = false
		}
	}

	return count
}

func countPacman() int {
	entries, err := os.ReadDir("/var/lib/pacman/local")
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "ALPM_DB_VERSION" {
			count++
		}
	}

	return count
}

func countRPM() int {
	out, err := exec.Command("rpm", "-qa").Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return len(lines)
}

func countEmerge() int {
	count := 0

	err := filepath.WalkDir("/var/db/pkg", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() && strings.Count(path, string(os.PathSeparator)) == 4 {
			count++
		}

		return nil
	})

	if err != nil {
		return 0
	}

	return count
}

func countFlatpak() int {
	count := 0

	systemPath := "/var/lib/flatpak/app"
	if entries, err := os.ReadDir(systemPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				count++
			}
		}
	}

	homeDir := os.Getenv("HOME")
	if homeDir != "" {
		userPath := filepath.Join(homeDir, ".local/share/flatpak/app")
		if entries, err := os.ReadDir(userPath); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					count++
				}
			}
		}
	}

	return count
}

func countSnap() int {
	entries, err := os.ReadDir("/snap")
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "bin" && entry.Name() != "README" {
			count++
		}
	}

	return count
}

func countNix() int {
	count := 0

	profilePaths := []string{
		"/nix/var/nix/profiles/system",
		filepath.Join(os.Getenv("HOME"), ".nix-profile"),
	}

	for _, profilePath := range profilePaths {
		manifestPath := filepath.Join(profilePath, "manifest.nix")
		if !pathExists(manifestPath) {
			manifestPath = filepath.Join(profilePath, "manifest.json")
		}

		if !pathExists(manifestPath) {
			continue
		}

		content, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.Contains(line, "name =") || strings.Contains(line, `"name":`) {
				count++
			}
		}
	}

	return count
}

func countBrew() int {
	cellarPaths := []string{
		"/opt/homebrew/Cellar",
		"/usr/local/Cellar",
	}

	for _, cellarPath := range cellarPaths {
		if entries, err := os.ReadDir(cellarPath); err == nil {
			return len(entries)
		}
	}

	return 0
}

func countPkgBSD() int {
	entries, err := os.ReadDir("/var/db/pkg")
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			count++
		}
	}

	return count
}

func formatPackageCount(count int, manager string) string {
	if manager == "" {
		return fmt.Sprintf("%d", count)
	}
	return fmt.Sprintf("%d (%s)", count, manager)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
