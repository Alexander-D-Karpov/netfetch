package model

type SystemInfo struct {
	OS           *OSInfo           `json:"os"`
	Host         string            `json:"host"`
	User         string            `json:"user"`
	Kernel       string            `json:"kernel"`
	Uptime       string            `json:"uptime"`
	Packages     string            `json:"packages"`
	Shell        string            `json:"shell"`
	Resolution   string            `json:"resolution"`
	DE           string            `json:"de"`
	WM           string            `json:"wm"`
	WMTheme      string            `json:"wm_theme"`
	Theme        string            `json:"theme"`
	Icons        string            `json:"icons"`
	Terminal     string            `json:"terminal"`
	CPU          *CPUInfo          `json:"cpu"`
	GPU          string            `json:"gpu"`
	Memory       *MemoryInfo       `json:"memory"`
	Disk         *DiskInfo         `json:"disk"`
	Network      *NetworkInfo      `json:"network"`
	Font         string            `json:"font"`
	Cursor       string            `json:"cursor"`
	TerminalFont string            `json:"terminal_font"`
	Swap         *SwapInfo         `json:"swap"`
	LocalIP      []string          `json:"local_ip"`
	Battery      *BatteryInfo      `json:"battery"`
	PowerAdapter *PowerAdapterInfo `json:"power_adapter"`
	Locale       string            `json:"locale"`
}

type OSInfo struct {
	Name       string `json:"name"`
	PrettyName string `json:"pretty_name"`
	Distro     string `json:"distro"`
	IDLike     string `json:"id_like"`
	Version    string `json:"version"`
	VersionID  string `json:"version_id"`
	Codename   string `json:"codename"`
	BuildID    string `json:"build_id"`
	Variant    string `json:"variant"`
	VariantID  string `json:"variant_id"`
	Arch       string `json:"arch"`
}

type CPUInfo struct {
	Name          string  `json:"name"`
	Vendor        string  `json:"vendor"`
	CoresPhysical uint16  `json:"cores_physical"`
	CoresLogical  uint16  `json:"cores_logical"`
	CoresOnline   uint16  `json:"cores_online"`
	FrequencyBase uint32  `json:"frequency_base"` // in MHz
	FrequencyMax  uint32  `json:"frequency_max"`  // in MHz
	Temperature   float64 `json:"temperature"`    // in Celsius
}

type MemoryInfo struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

type DiskInfo struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type NetworkInfo struct {
	Interfaces []InterfaceInfo `json:"interfaces"`
}

type InterfaceInfo struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

type SwapInfo struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

type BatteryInfo struct {
	Percentage float64 `json:"percentage"`
	Status     string  `json:"status"`
}

type PowerAdapterInfo struct {
	IsConnected bool `json:"is_connected"`
}
