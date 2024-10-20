package collector

func (c *Collector) collectResolution() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.Resolution = "1920x1080, 3440x1440" // TODO
	return c.info.Resolution
}
