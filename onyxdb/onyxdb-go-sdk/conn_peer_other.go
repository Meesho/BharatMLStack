//go:build !linux && !darwin

package sdk

// peerOpen cannot peek at the socket on this platform, so a held connection is
// taken as open; a dead one fails its next request and is closed then.
func (c *Conn) peerOpen() bool { return true }
