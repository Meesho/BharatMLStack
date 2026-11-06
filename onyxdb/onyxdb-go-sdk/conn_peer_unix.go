//go:build linux || darwin

package sdk

import "syscall"

// peerOpen reports whether the peer still holds an idle connection open. It
// peeks without blocking or consuming: an open connection at a request
// boundary has nothing to read (EAGAIN), while a FIN reads as 0 bytes, a reset
// as an error, and bytes nobody asked for leave the stream out of step, so all
// three count as closed. A connection without a socket counts as open.
func (c *Conn) peerOpen() bool {
	sc, ok := c.conn.(syscall.Conn)
	if !ok {
		return true
	}
	open := false
	raw, err := sc.SyscallConn()
	if err == nil {
		err = raw.Read(func(fd uintptr) bool {
			var b [1]byte
			_, _, rerr := syscall.Recvfrom(int(fd), b[:], syscall.MSG_PEEK|syscall.MSG_DONTWAIT)
			open = rerr == syscall.EAGAIN
			return true // never wait for readability
		})
	}
	return err == nil && open
}
