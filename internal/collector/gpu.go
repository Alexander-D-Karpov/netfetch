package collector

func (c *Collector) collectGPU() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.info.GPU = "NVIDIA GeForce RTX 3060 Lite Hash Rate" // TODO
	return c.info.GPU
}
