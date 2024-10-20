package collector

func (c *Collector) collectDE() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.DE = "Plasma 6.1.4" // TODO
	return c.info.DE
}
