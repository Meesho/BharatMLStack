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

// Cache is the minimal interface required by the RESP server.
// Implementations should be safe for concurrent use.
type Cache interface {
	Put(key string, value []byte, ttl time.Duration) error
	Get(key string) ([]byte, bool, bool)
}

// ServeRESP starts a minimal RESP (Redis) protocol server over TCP supporting
// GET and SET only. It is optimized for low overhead and pipelined requests.
//
// Supported commands (case-insensitive):
//   - GET key
//   - SET key value [EX seconds]
func ServeRESP(addr string, cache Cache) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			return err
		}
		if tc, ok := conn.(*net.TCPConn); ok {
			_ = tc.SetNoDelay(true)
			_ = tc.SetKeepAlive(true)
			_ = tc.SetKeepAlivePeriod(3 * time.Minute)
		}
		go handleConn(conn, cache)
	}
}

func handleConn(conn net.Conn, cache Cache) {
	defer conn.Close()
	r := bufio.NewReaderSize(conn, 64*1024)
	w := bufio.NewWriterSize(conn, 64*1024)

	for {
		cmd, args, perr := readRESPArray(r)
		if perr != nil {
			if perr == io.EOF || errors.Is(perr, net.ErrClosed) {
				return
			}
			return
		}
		if len(cmd) == 0 {
			continue
		}

		switch {
		case len(cmd) == 3 && (cmd[0]|0x20) == 'g' && (cmd[1]|0x20) == 'e' && (cmd[2]|0x20) == 't':
			if len(args) != 1 {
				writeError(w, "wrong number of arguments for 'get'")
			} else {
				val, found, expired := cache.Get(b2s(args[0]))
				if !found || expired {
					writeBulkNil(w)
				} else {
					writeBulk(w, val)
				}
			}

		case len(cmd) >= 3 && (cmd[0]|0x20) == 's' && (cmd[1]|0x20) == 'e' && (cmd[2]|0x20) == 't':
			if len(args) != 2 && len(args) != 4 {
				writeError(w, "wrong number of arguments for 'set'")
			} else {
				key := b2s(args[0])
				value := args[1]
				var ttl time.Duration
				if len(args) == 4 {
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
					ttl = time.Duration(secs) * time.Second
				}
				_ = cache.Put(key, value, ttl)
				writeSimpleString(w, "OK")
			}

		default:
			writeError(w, "unknown command")
		}

		if w.Flush() != nil {
			return
		}
	}
}

func readRESPArray(r *bufio.Reader) (cmd []byte, args [][]byte, err error) {
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
	bs, err := readBulkString(r)
	if err != nil {
		return nil, nil, err
	}
	cmd = bs
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
		return nil, nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	if err := expectCRLF(r); err != nil {
		return nil, err
	}
	return buf, nil
}

func readIntCRLF(r *bufio.Reader) (int, error) {
	line, err := r.ReadSlice('\r')
	if err != nil {
		return 0, err
	}
	if b, err := r.ReadByte(); err != nil || b != '\n' {
		if err == nil {
			err = io.ErrUnexpectedEOF
		}
		return 0, err
	}
	line = line[:len(line)-1]
	return strconv.Atoi(b2s(line))
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

func b2s(b []byte) string                { return string(b) }
func parseUint(b []byte) (uint64, error) { return strconv.ParseUint(string(b), 10, 64) }
