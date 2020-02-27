// Package net provides a portable interface for network I/O. It is extensible:
// you can provide your own network interface that this package will use if none
// was already defined by the system.
package net

// adapter is the adapter that was last set by SetAdapter, and should be used by
// all new network operations.
var adapter Adapter

// Adapter represents a low-level network interface driver. It can be either
// implemented by the host OS, be a simulated driver (for testing), or it can be
// a hardware peripheral or even on-chip WiFi if available.
type Adapter interface {
	Dial(network, address string) (Conn, error)
}

// SetAdapter sets or replaces the current network driver. All newly created
// connections will use the new driver, old connections will continue to use the
// existing driver.
func SetAdapter(a Adapter) {
	adapter = a
}

// Conn is a generic stream-oriented network connection.
type Conn interface {
	// Read reads data from the connection.
	Read(b []byte) (n int, err error)

	// Write writes data to the connection.
	Write(b []byte) (n int, err error)

	// Close closes the connection.
	Close() error
}

// Dial connects to the address on the named network.
func Dial(network, address string) (Conn, error) {
	return adapter.Dial(network, address)
}
