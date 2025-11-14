package handler

import (
	"log"
	"net/http"
	"strings"

	"netfetch/internal/collector"
	"netfetch/internal/config"
	"netfetch/internal/logo"
)

type Handler struct {
	collector *collector.Collector
	logos     map[string]*logo.Logo
	config    *config.Config
}

func New(c *collector.Collector, l map[string]*logo.Logo, cfg *config.Config) *Handler {
	return &Handler{
		collector: c,
		logos:     l,
		config:    cfg,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.collector.CollectDynamicInfo()
	if strings.Contains(r.Header.Get("User-Agent"), "curl") {
		h.handleCurl(w)
	} else {
		h.handleWeb(w)
	}
}

func (h *Handler) getLogo(distro string) *logo.Logo {
	if distro == "" {
		distro = h.config.DefaultLogo
	}

	distroLower := strings.ToLower(distro)
	distroLower = strings.ReplaceAll(distroLower, " ", "_")
	distroLower = strings.ReplaceAll(distroLower, "-", "_")

	tryNames := []string{distroLower}

	distroVariations := map[string][]string{
		"ubuntu": {
			"ubuntu", "ubuntu_small", "ubuntu_old",
		},
		"kubuntu": {
			"kubuntu", "ubuntu", "ubuntu_small",
		},
		"xubuntu": {
			"xubuntu", "ubuntu", "ubuntu_small",
		},
		"lubuntu": {
			"lubuntu", "ubuntu", "ubuntu_small",
		},
		"ubuntu_budgie": {
			"ubuntu_budgie", "ubuntu", "ubuntu_small",
		},
		"ubuntu_cinnamon": {
			"ubuntu_cinnamon", "ubuntu", "ubuntu_small",
		},
		"ubuntu_gnome": {
			"ubuntu_gnome", "ubuntu", "ubuntu_small",
		},
		"ubuntu_mate": {
			"ubuntu_mate", "ubuntu", "ubuntu_small",
		},
		"ubuntu_studio": {
			"ubuntu_studio", "ubuntu", "ubuntu_small",
		},
		"arch": {
			"arch", "archlinux", "arch_small", "arch_old",
		},
		"archlinux": {
			"arch", "archlinux", "arch_small", "arch_old",
		},
		"manjaro": {
			"manjaro", "manjaro_small", "arch", "arch_small",
		},
		"endeavouros": {
			"endeavouros", "arch", "arch_small",
		},
		"arcolinux": {
			"arcolinux", "arcolinux_small", "arch", "arch_small",
		},
		"debian": {
			"debian", "debian_small",
		},
		"fedora": {
			"fedora", "fedora_small",
		},
		"centos": {
			"centos", "centos_small", "rhel", "rhel_old",
		},
		"rhel": {
			"rhel", "rhel_old", "centos", "centos_small",
		},
		"rocky": {
			"rocky", "rocky_small", "rhel", "centos",
		},
		"almalinux": {
			"almalinux", "rhel", "centos",
		},
		"opensuse": {
			"opensuse_tumbleweed", "opensuse_leap", "suse", "suse_small",
		},
		"suse": {
			"suse", "suse_small", "opensuse_tumbleweed", "opensuse_leap",
		},
		"gentoo": {
			"gentoo", "gentoo_small",
		},
		"alpine": {
			"alpine", "alpine_small",
		},
		"void": {
			"void", "void_small",
		},
		"nixos": {
			"nixos", "nixos_small", "nixos_old",
		},
		"freebsd": {
			"freebsd", "freebsd_small", "bsd",
		},
		"openbsd": {
			"openbsd", "openbsd_small", "bsd",
		},
		"netbsd": {
			"netbsd", "netbsd_small", "bsd",
		},
		"dragonfly": {
			"dragonfly", "dragonfly_small", "dragonfly_old", "bsd",
		},
		"mint": {
			"mint", "linuxmint_small", "mint_old", "debian", "ubuntu",
		},
		"linuxmint": {
			"mint", "linuxmint_small", "mint_old", "debian", "ubuntu",
		},
		"mx": {
			"mx", "mx_small", "debian",
		},
		"pop": {
			"pop_os", "pop_os_small", "ubuntu", "debian",
		},
		"pop_os": {
			"pop_os", "pop_os_small", "ubuntu", "debian",
		},
		"elementary": {
			"elementary", "elementary_small", "ubuntu", "debian",
		},
		"zorin": {
			"zorin", "ubuntu", "debian",
		},
		"kali": {
			"kali", "debian",
		},
		"parrot": {
			"parrot", "debian",
		},
		"raspbian": {
			"raspbian", "raspbian_small", "debian",
		},
		"solus": {
			"solus",
		},
		"mageia": {
			"mageia", "mageia_small",
		},
		"slackware": {
			"slackware", "slackware_small",
		},
		"windows": {
			"windows", "windows11", "windows8",
		},
		"macos": {
			"darwin",
		},
		"darwin": {
			"darwin",
		},
	}

	if variations, exists := distroVariations[distroLower]; exists {
		tryNames = variations
	} else {
		tryNames = append(tryNames, distroLower+"_small")
	}

	for _, name := range tryNames {
		if logo, ok := h.logos[name]; ok {
			if name != distroLower {
				log.Printf("Using logo '%s' for distro '%s'", name, distro)
			}
			return logo
		}
	}

	distroFamilies := map[string][]string{
		"ubuntu":    {"debian", "linux"},
		"debian":    {"linux"},
		"arch":      {"linux"},
		"fedora":    {"rhel", "linux"},
		"centos":    {"rhel", "linux"},
		"rhel":      {"linux"},
		"opensuse":  {"suse", "linux"},
		"suse":      {"linux"},
		"gentoo":    {"linux"},
		"alpine":    {"linux"},
		"void":      {"linux"},
		"nixos":     {"linux"},
		"slackware": {"linux"},
		"freebsd":   {"bsd"},
		"openbsd":   {"bsd"},
		"netbsd":    {"bsd"},
		"dragonfly": {"bsd"},
	}

	if families, exists := distroFamilies[distroLower]; exists {
		for _, family := range families {
			if logo, ok := h.logos[family]; ok {
				log.Printf("Using family logo '%s' for distro '%s'", family, distro)
				return logo
			}
		}
	}

	if h.config.DefaultLogo != "" && h.config.DefaultLogo != distro {
		defaultLower := strings.ToLower(h.config.DefaultLogo)
		if logo, ok := h.logos[defaultLower]; ok {
			log.Printf("Using default logo '%s' for distro '%s'", defaultLower, distro)
			return logo
		}
	}

	genericLogos := []string{"linux", "gnu", "bsd", "unix"}
	for _, generic := range genericLogos {
		if logo, ok := h.logos[generic]; ok {
			log.Printf("Using generic logo '%s' for distro '%s'", generic, distro)
			return logo
		}
	}

	if len(h.logos) > 0 {
		for name, logo := range h.logos {
			log.Printf("Using fallback logo '%s' for distro '%s'", name, distro)
			return logo
		}
	}

	log.Printf("Warning: No logos available for distro '%s'", distro)
	return nil
}
