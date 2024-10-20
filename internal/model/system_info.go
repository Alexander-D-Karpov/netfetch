package model

type SystemInfo struct {
	OS         *OSInfo      `json:"os"`
	Host       string       `json:"host"`
	Kernel     string       `json:"kernel"`
	Uptime     string       `json:"uptime"`
	Packages   string       `json:"packages"`
	Shell      string       `json:"shell"`
	Resolution string       `json:"resolution"`
	DE         string       `json:"de"`
	WM         string       `json:"wm"`
	WMTheme    string       `json:"wm_theme"`
	Theme      string       `json:"theme"`
	Icons      string       `json:"icons"`
	Terminal   string       `json:"terminal"`
	CPU        *CPUInfo     `json:"cpu"`
	GPU        string       `json:"gpu"`
	Memory     *MemoryInfo  `json:"memory"`
	Disk       *DiskInfo    `json:"disk"`
	Network    *NetworkInfo `json:"network"`
}

type OSInfo struct {
	Name   string `json:"name"`
	Distro string `json:"distro"`
	Arch   string `json:"arch"`
}

type CPUInfo struct {
	Model     string  `json:"model"`
	Cores     int     `json:"cores"`
	Frequency float64 `json:"frequency"`
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
