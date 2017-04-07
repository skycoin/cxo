package gnet

// Message represents interface that every
// message must implement to be handled
type Message interface {
	// Handle called from remote side
	// when the Message received. If
	// the Handle returns an error then
	// the connections will be terminated
	// using the error as reason to the
	// termination. The user argument is
	// any interface{} provided to Pool
	// as user data
	Handle(ctx MessageContext, user interface{}) (terminate error)
}

type MessageContext interface {
	// Send reply back to connection
	// from which the message received
	Send(m Message)
	// Broadcast to all connections except
	// connection from which the message
	// received
	Broadcast(m Message)
	// Addr returns remote address
	Addr() string
}

// receivedMessage represents received and
// decoded message that ready to be handled
// the receivedMessage implements MessageContext
type receivedMessage struct {
	*Conn
	msg Message
}
