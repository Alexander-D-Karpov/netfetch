package collector

func (c *Collector) collectWM() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.WM = "KWin"                        // TODO
	c.info.WMTheme = "Sweet-Mars-transparent" // TODO
	return c.info.WM
}
