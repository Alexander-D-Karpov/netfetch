package collector

func (c *Collector) collectTheme() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Theme = "[Plasma], Breeze-Dark [GTK2], Breeze [GTK3]" // TODO
	return c.info.Theme
}
