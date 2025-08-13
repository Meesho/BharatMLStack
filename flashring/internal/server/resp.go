package server

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"strconv"
	"time"
)

// KV is the minimal cache interface required by the RESP server.
// Implementations should be safe for concurrent use.
type KV interface {
	// Put stores the value with optional expire time in unix seconds (0 for no expiry).
	Put(key string, value []byte, exptime uint64) error
	// Get returns value, keyFound, expired
	Get(key string) ([]byte, bool, bool)
}

// ServeRESP starts a minimal RESP (Redis) protocol server over TCP supporting
// GET and SET only. It is optimized for low overhead and pipelined requests.
//
// Supported commands (case-insensitive):
//   - *2\r\n$3\r\nGET\r\n$<klen>\r\n<key>\r\n
//   - *3\r\n$3\r\nSET\r\n$<klen>\r\n<key>\r\n$<vlen>\r\n<val>\r\n
//   - SET with EX seconds (optional):
//     *5 ... SET key val EX seconds
//
// Inline protocol is not supported to keep parsing fast and simple.
func ServeRESP(addr string, cache KV) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	// Accept loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			return err
		}
		// Configure TCP for low latency
		if tc, ok := conn.(*net.TCPConn); ok {
			_ = tc.SetNoDelay(true)
			_ = tc.SetKeepAlive(true)
			_ = tc.SetKeepAlivePeriod(3 * time.Minute)
		}
		go handleConn(conn, cache)
	}
}

func handleConn(conn net.Conn, cache KV) {
	defer conn.Close()
	// Generous buffers for pipelining
	r := bufio.NewReaderSize(conn, 64*1024)
	w := bufio.NewWriterSize(conn, 64*1024)
	for {
		cmd, args, perr := readRESPArray(r)
		if perr != nil {
			if perr == io.EOF || errors.Is(perr, net.ErrClosed) {
				return
			}
			// Protocol error: close connection per Redis behavior
			return
		}
		if len(cmd) == 0 {
			// Ignore empty
			continue
		}
		// Fast upper-case compare for GET/SET without heap allocs
		if len(cmd) == 3 && (cmd[0]|0x20) == 'g' && (cmd[1]|0x20) == 'e' && (cmd[2]|0x20) == 't' {
			// GET key
			if len(args) != 1 {
				writeError(w, "wrong number of arguments for 'get'")
				if w.Flush() != nil {
					return
				}
				continue
			}
			key := b2s(args[0])
			val, found, expired := cache.Get(key)
			if !found || expired {
				writeBulkNil(w)
			} else {
				writeBulk(w, val)
			}
			if w.Flush() != nil {
				return
			}
			continue
		}
		if len(cmd) >= 3 && (cmd[0]|0x20) == 's' && (cmd[1]|0x20) == 'e' && (cmd[2]|0x20) == 't' {
			// SET key value [EX seconds]
			if len(args) != 2 && len(args) != 4 {
				writeError(w, "wrong number of arguments for 'set'")
				if w.Flush() != nil {
					return
				}
				continue
			}
			key := b2s(args[0])
			value := args[1]
			var ex uint64
			if len(args) == 4 {
				// Expect EX seconds
				if !bytes.EqualFold(args[2], []byte("EX")) {
					writeError(w, "only EX option is supported")
					if w.Flush() != nil {
						return
					}
					continue
				}
				secs, err := parseUint(args[3])
				if err != nil {
					writeError(w, "invalid expire seconds")
					if w.Flush() != nil {
						return
					}
					continue
				}
				ex = secs
			}
			_ = cache.Put(key, value, ex)
			writeSimpleString(w, "OK")
			if w.Flush() != nil {
				return
			}
			continue
		}
		// Unknown command
		writeError(w, "unknown command")
		if w.Flush() != nil {
			return
		}
	}
}

// RESP helpers

// readRESPArray parses a RESP Array of Bulk Strings and returns command and args.
// It assumes arrays consisting only of bulk strings; inline protocol is not supported.
func readRESPArray(r *bufio.Reader) (cmd []byte, args [][]byte, err error) {
	// Expect '*'
	b, err := r.ReadByte()
	if err != nil {
		return nil, nil, err
	}
	if b != '*' {
		return nil, nil, io.ErrUnexpectedEOF
	}
	n, err := readIntCRLF(r)
	if err != nil {
		return nil, nil, err
	}
	if n <= 0 {
		return nil, nil, nil
	}
	// First element is command
	bs, err := readBulkString(r)
	if err != nil {
		return nil, nil, err
	}
	cmd = bs
	// Remaining are args
	if n > 1 {
		args = make([][]byte, 0, n-1)
		for i := 1; i < n; i++ {
			bsi, err := readBulkString(r)
			if err != nil {
				return nil, nil, err
			}
			args = append(args, bsi)
		}
	}
	return
}

func readBulkString(r *bufio.Reader) ([]byte, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if b != '$' {
		return nil, io.ErrUnexpectedEOF
	}
	n, err := readIntCRLF(r)
	if err != nil {
		return nil, err
	}
	if n < 0 {
		// Null bulk string
		return nil, nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	// Read trailing CRLF
	if err := expectCRLF(r); err != nil {
		return nil, err
	}
	return buf, nil
}

func readIntCRLF(r *bufio.Reader) (int, error) {
	// Read until CR
	line, err := r.ReadSlice('\r')
	if err != nil {
		return 0, err
	}
	// Next must be '\n'
	if b, err := r.ReadByte(); err != nil || b != '\n' {
		if err == nil {
			err = io.ErrUnexpectedEOF
		}
		return 0, err
	}
	// Trim trailing CR
	line = line[:len(line)-1]
	// Parse signed/unsigned int
	// Use strconv for correctness; line is small
	i, err := strconv.Atoi(b2s(line))
	if err != nil {
		return 0, err
	}
	return i, nil
}

func expectCRLF(r *bufio.Reader) error {
	c1, err := r.ReadByte()
	if err != nil {
		return err
	}
	c2, err := r.ReadByte()
	if err != nil {
		return err
	}
	if c1 != '\r' || c2 != '\n' {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func writeSimpleString(w *bufio.Writer, s string) {
	w.WriteByte('+')
	w.WriteString(s)
	w.WriteString("\r\n")
}

func writeError(w *bufio.Writer, s string) {
	w.WriteByte('-')
	w.WriteString("ERR ")
	w.WriteString(s)
	w.WriteString("\r\n")
}

func writeBulk(w *bufio.Writer, p []byte) {
	w.WriteByte('$')
	w.WriteString(strconv.Itoa(len(p)))
	w.WriteString("\r\n")
	w.Write(p)
	w.WriteString("\r\n")
}

func writeBulkNil(w *bufio.Writer) {
	w.WriteString("$-1\r\n")
}

// b2s converts []byte to string with allocation.
// We intentionally avoid unsafe tricks for portability.
func b2s(b []byte) string                { return string(b) }
func parseUint(b []byte) (uint64, error) { return strconv.ParseUint(string(b), 10, 64) }
