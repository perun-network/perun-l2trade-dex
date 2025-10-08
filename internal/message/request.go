package message

func (c *Connection) nextRequestId() uint64 {
	c.requestCounterMu.Lock()
	defer c.requestCounterMu.Unlock()
	c.requestCounter++
	return c.requestCounter
}

// Request wraps the `msg` in a Request and returns the Response `resp` sent by
// the client as well as the used `id` to match them.
func (c *Connection) Request(msg Message) (resp Message, err error) {
	id := c.nextRequestId()

	response := make(chan Message)
	c.responseHandlers.set(id, response)
	defer c.responseHandlers.delete(id)

	req := NewRequest(id, msg)
	err = c.Write(req)
	if err != nil {
		return
	}

	resp = <-response
	return
}
