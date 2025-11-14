package model

type SystemInfo struct {
	OS            *OSInfo           `json:"os"`
	Host          string            `json:"host"`
	User          string            `json:"user"`
	Kernel        string            `json:"kernel"`
	Uptime        string            `json:"uptime"`
	Packages      string            `json:"packages"`
	Shell         string            `json:"shell"`
	Resolution    string            `json:"resolution"`
	DE            string            `json:"de"`
	WM            string            `json:"wm"`
	WMTheme       string            `json:"wm_theme"`
	Theme         string            `json:"theme"`
	Icons         string            `json:"icons"`
	Terminal      string            `json:"terminal"`
	CPU           *CPUInfo          `json:"cpu"`
	GPU           string            `json:"gpu"`
	GPUTemp       int               `json:"gpu_temp"`
	Memory        *MemoryInfo       `json:"memory"`
	Disk          *DiskInfo         `json:"disk"`
	Disks         []DiskInfo        `json:"disks"`
	PhysicalDisks []PhysicalDisk    `json:"physical_disks"`
	Network       *NetworkInfo      `json:"network"`
	Font          string            `json:"font"`
	Cursor        string            `json:"cursor"`
	TerminalFont  string            `json:"terminal_font"`
	Swap          *SwapInfo         `json:"swap"`
	LocalIP       []string          `json:"local_ip"`
	Battery       *BatteryInfo      `json:"battery"`
	PowerAdapter  *PowerAdapterInfo `json:"power_adapter"`
	Locale        string            `json:"locale"`
	HostInfo      *HostInfo         `json:"host_info"`
	BIOS          *BIOSInfo         `json:"bios"`
	Processes     int               `json:"processes"`
	CPUUsage      float64           `json:"cpu_usage"`
	PublicIP      string            `json:"public_ip"`
	Wifi          *WifiInfo         `json:"wifi"`
	DateTime      string            `json:"datetime"`
	Users         []UserInfo        `json:"users"`
	Brightness    *BrightnessInfo   `json:"brightness"`
	LoginManager  string            `json:"login_manager"`
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
	FrequencyBase uint32  `json:"frequency_base"`
	FrequencyMax  uint32  `json:"frequency_max"`
	Temperature   float64 `json:"temperature"`
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

	Mountpoint string `json:"mountpoint,omitempty"`
	FSType     string `json:"fs_type,omitempty"`
	Device     string `json:"device,omitempty"`
	Label      string `json:"label,omitempty"`
}

type PhysicalDisk struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	Size       uint64 `json:"size"`
	Type       string `json:"type"`
	Rotational bool   `json:"rotational"`
}

type NetworkInfo struct {
	Interfaces []InterfaceInfo `json:"interfaces"`
}

type InterfaceInfo struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

type SwapInfo struct {
	Total  uint64 `json:"total"`
	Used   uint64 `json:"used"`
	Free   uint64 `json:"free"`
	Device string `json:"device"`
}

type BatteryInfo struct {
	Percentage float64 `json:"percentage"`
	Status     string  `json:"status"`
}

type PowerAdapterInfo struct {
	IsConnected bool `json:"is_connected"`
}

type HostInfo struct {
	Vendor  string `json:"vendor"`
	Model   string `json:"model"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

type BIOSInfo struct {
	Vendor  string `json:"vendor"`
	Version string `json:"version"`
	Date    string `json:"date"`
	Type    string `json:"type"`
}

type WifiInfo struct {
	SSID      string `json:"ssid"`
	Protocol  string `json:"protocol"`
	Frequency string `json:"frequency"`
	Security  string `json:"security"`
	Strength  int    `json:"strength"`
}

type UserInfo struct {
	Name      string `json:"name"`
	Terminal  string `json:"terminal"`
	LoginTime string `json:"login_time"`
}

type BrightnessInfo struct {
	Current int `json:"current"`
	Max     int `json:"max"`
}
