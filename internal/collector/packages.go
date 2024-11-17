package collector

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type PackageManager struct {
	Name    string            // Name of the package manager
	Command string            // Command to execute
	Args    []string          // Arguments for package counting
	Offset  int               // Number to subtract from count (e.g., for header lines)
	Filter  func(string) bool // Optional function to filter package lines
}

var packageManagers = []PackageManager{
	{
		Name:    "pacman",
		Command: "pacman",
		Args:    []string{"-Qq"},
		Offset:  1, // For trailing newline
	},
	{
		Name:    "dpkg",
		Command: "dpkg-query",
		Args:    []string{"-f", "${binary:Package}\n", "-W"},
		Offset:  1,
	},
	{
		Name:    "rpm",
		Command: "rpm",
		Args:    []string{"-qa", "--queryformat", "%{NAME}\n"},
		Offset:  1,
	},
	{
		Name:    "flatpak",
		Command: "flatpak",
		Args:    []string{"list"},
		Offset:  2, // Header line and trailing newline
		Filter: func(line string) bool {
			return len(line) > 0 && !strings.HasPrefix(line, "Application") && !strings.HasPrefix(line, "Runtime")
		},
	},
	{
		Name:    "snap",
		Command: "snap",
		Args:    []string{"list"},
		Offset:  2, // Header line and trailing newline
		Filter: func(line string) bool {
			return len(line) > 0 && !strings.HasPrefix(line, "Name")
		},
	},
	{
		Name:    "brew",
		Command: "brew",
		Args:    []string{"list", "--formula"},
		Offset:  1,
	},
	{
		Name:    "brew-cask",
		Command: "brew",
		Args:    []string{"list", "--cask"},
		Offset:  1,
	},
	{
		Name:    "apk",
		Command: "apk",
		Args:    []string{"info"},
		Offset:  1,
	},
	{
		Name:    "xbps",
		Command: "xbps-query",
		Args:    []string{"-l"},
		Offset:  1,
		Filter: func(line string) bool {
			return len(line) > 0 && strings.HasPrefix(line, "ii")
		},
	},
	{
		Name:    "opkg",
		Command: "opkg",
		Args:    []string{"list-installed"},
		Offset:  1,
	},
}

// PackageCount represents the count from a specific package manager
type PackageCount struct {
	Manager string
	Count   int
	Error   error
}

func (c *Collector) collectPackages() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.activeModules["packages"] {
		c.info.Packages = "Unknown"
		return
	}

	counts := make(chan PackageCount, len(packageManagers))
	var wg sync.WaitGroup

	// Start a goroutine for each package manager
	for _, pm := range packageManagers {
		wg.Add(1)
		go func(pm PackageManager) {
			defer wg.Done()
			count, err := countPackages(pm)
			counts <- PackageCount{
				Manager: pm.Name,
				Count:   count,
				Error:   err,
			}
		}(pm)
	}

	// Close counts channel after all goroutines complete
	go func() {
		wg.Wait()
		close(counts)
	}()

	// Collect results
	var totalPackages int
	var details []string
	seenManagers := make(map[string]bool)

	for count := range counts {
		if count.Error != nil || count.Count == 0 {
			continue
		}

		// Skip duplicate manager names (e.g., brew formula and cask)
		baseManager := strings.Split(count.Manager, "-")[0]
		if seenManagers[baseManager] {
			totalPackages += count.Count
			continue
		}
		seenManagers[baseManager] = true

		totalPackages += count.Count
		details = append(details, fmt.Sprintf("%d (%s)", count.Count, count.Manager))
	}

	if totalPackages > 0 {
		c.info.Packages = fmt.Sprintf("%d (%s)", totalPackages, strings.Join(details, ", "))
	} else {
		c.info.Packages = "Unknown"
	}
}

func countPackages(pm PackageManager) (int, error) {
	// Check if the command exists
	path, err := exec.LookPath(pm.Command)
	if err != nil {
		return 0, fmt.Errorf("command not found: %s", pm.Command)
	}

	// Execute the command
	cmd := exec.Command(path, pm.Args...)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("error executing %s: %v", pm.Command, err)
	}

	// Split output into lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Apply filter if provided
	if pm.Filter != nil {
		filteredLines := make([]string, 0, len(lines))
		for _, line := range lines {
			if pm.Filter(line) {
				filteredLines = append(filteredLines, line)
			}
		}
		lines = filteredLines
	}

	// Calculate count
	count := len(lines)
	if count > 0 {
		count -= pm.Offset
	}
	if count < 0 {
		count = 0
	}

	return count, nil
}
