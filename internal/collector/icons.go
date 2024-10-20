package collector

func (c *Collector) collectIcons() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Icons = "candy-icons [Plasma], candy-icons [GTK2/3]" // TODO
	return c.info.Icons
}
