package collector

func (c *Collector) collectTerminal() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Terminal = "alacritty" // TODO
	return c.info.Terminal
}
