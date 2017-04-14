package gnet

// Message represent a data received from a connection
type Message struct {
	// Value keeps encoded data that received
	Value interface{}
	// Conn is connection from witch the data comes from
	Conn *Conn
}
