package client

func Example_usage() {
	c, err := NewClient()
	if err != nil {
		// handle error
		return
	}
	if err = c.Start(); err != nil {
		// handle error
		return
	}
	defer c.Close()
	// waiting for SIGINT (Ctrl+C)
	c.WaitInterrupt()
}
