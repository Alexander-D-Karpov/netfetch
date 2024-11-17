package collector

func (c *Collector) collectDE() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.DE = getDE()
}

func getDE() string {
	deProcesses := map[string][]string{
		"GNOME":    {"gnome-session", "gnome-shell"},
		"KDE":      {"plasmashell", "ksmserver"},
		"XFCE":     {"xfce4-session"},
		"Cinnamon": {"cinnamon-session"},
		"MATE":     {"mate-session"},
		"Unity":    {"unity-panel-service"},
		"LXDE":     {"lxsession"},
		"Deepin":   {"dde-desktop"},
		"Pantheon": {"gala"},
		"Budgie":   {"budgie-wm"},
		"LXQt":     {"lxqt-session"},
	}

	return detectProcess(deProcesses)
}
