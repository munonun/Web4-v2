package transport

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"web4/protocol"
)

const maxHandshakeAddrLen = 1024

type TCPTransport struct {
	listenAddr string

	mu       sync.Mutex
	handler  Handler
	listener net.Listener
	closed   bool
	started  bool
	conns    map[string]*tcpConn
	wg       sync.WaitGroup
}

type tcpConn struct {
	conn      net.Conn
	writeMu   sync.Mutex
	closeOnce sync.Once

	mu    sync.Mutex
	keys  map[string]struct{}
	peer  string
	ready chan struct{}
	once  sync.Once

	handshakeErr error
}

func NewTCPTransport(listenAddr string) *TCPTransport {
	return &TCPTransport{
		listenAddr: listenAddr,
		conns:      make(map[string]*tcpConn),
	}
}

func (t *TCPTransport) Addr() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.listener != nil {
		return t.listener.Addr().String()
	}
	return t.listenAddr
}

func (t *TCPTransport) Start(handler Handler) error {
	if handler == nil {
		return fmt.Errorf("nil handler")
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("transport closed")
	}
	if t.started {
		return fmt.Errorf("transport already started")
	}

	listener, err := net.Listen("tcp", t.listenAddr)
	if err != nil {
		return err
	}

	t.listener = listener
	t.handler = handler
	t.started = true
	t.wg.Add(1)
	go t.acceptLoop(listener)
	return nil
}

func (t *TCPTransport) Send(peer string, msg protocol.Message) error {
	frame := EncodeMessageFrame(msg)

	conn, err := t.getOrDial(peer)
	if err != nil {
		return err
	}
	if err := conn.waitReady(); err != nil {
		return err
	}

	if err := conn.write(frame); err != nil {
		t.dropConn(conn)
		return err
	}
	return nil
}

func (t *TCPTransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	listener := t.listener
	seen := make(map[*tcpConn]struct{})
	conns := make([]*tcpConn, 0, len(t.conns))
	for _, conn := range t.conns {
		if _, ok := seen[conn]; ok {
			continue
		}
		seen[conn] = struct{}{}
		conns = append(conns, conn)
	}
	t.mu.Unlock()

	var closeErr error
	if listener != nil {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			closeErr = err
		}
	}
	for _, conn := range conns {
		if err := conn.close(); err != nil && closeErr == nil && !errors.Is(err, net.ErrClosed) {
			closeErr = err
		}
	}
	t.wg.Wait()
	return closeErr
}

func (t *TCPTransport) acceptLoop(listener net.Listener) {
	defer t.wg.Done()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if t.isClosed() || errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		t.startConn(conn, false, "")
	}
}

func (t *TCPTransport) getOrDial(peer string) (*tcpConn, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport closed")
	}
	if !t.started {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport not started")
	}
	if conn, ok := t.conns[peer]; ok {
		t.mu.Unlock()
		return conn, nil
	}
	t.mu.Unlock()

	netConn, err := net.Dial("tcp", peer)
	if err != nil {
		return nil, err
	}
	return t.startConn(netConn, true, peer), nil
}

func (t *TCPTransport) startConn(netConn net.Conn, outbound bool, peer string) *tcpConn {
	conn := &tcpConn{
		conn:  netConn,
		keys:  make(map[string]struct{}),
		ready: make(chan struct{}),
	}
	if outbound {
		conn.addKey(peer)
		t.mu.Lock()
		if existing, ok := t.conns[peer]; ok && existing != conn {
			t.mu.Unlock()
			_ = netConn.Close()
			return existing
		}
		t.conns[peer] = conn
		t.mu.Unlock()
		if err := conn.write(encodeHandshake(t.Addr())); err != nil {
			conn.failReady(err)
			t.dropConn(conn)
			return conn
		}
	}

	t.wg.Add(1)
	go t.readLoop(conn, outbound)
	return conn
}

func (t *TCPTransport) readLoop(conn *tcpConn, outbound bool) {
	defer t.wg.Done()
	defer t.dropConn(conn)

	if !outbound {
		if err := conn.write(encodeHandshake(t.Addr())); err != nil {
			conn.failReady(err)
			return
		}
	}

	peer, err := decodeHandshake(conn.conn)
	if err != nil {
		conn.failReady(err)
		return
	}
	conn.setPeer(peer)
	t.registerConnKey(conn, peer)
	conn.markReady()

	for {
		msg, err := DecodeMessageFrame(conn.conn)
		if err != nil {
			return
		}
		if handler := t.getHandler(); handler != nil {
			_ = handler(peer, msg)
		}
	}
}

func (t *TCPTransport) getHandler() Handler {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.handler
}

func (t *TCPTransport) registerConnKey(conn *tcpConn, key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	if existing, ok := t.conns[key]; ok && existing != conn {
		return
	}
	conn.addKey(key)
	t.conns[key] = conn
}

func (t *TCPTransport) dropConn(conn *tcpConn) {
	t.mu.Lock()
	for key := range conn.keysSnapshot() {
		if t.conns[key] == conn {
			delete(t.conns, key)
		}
	}
	t.mu.Unlock()
	_ = conn.close()
	conn.failReady(io.EOF)
}

func (t *TCPTransport) isClosed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

func (c *tcpConn) write(frame []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	for len(frame) > 0 {
		n, err := c.conn.Write(frame)
		if err != nil {
			return err
		}
		frame = frame[n:]
	}
	return nil
}

func (c *tcpConn) close() error {
	var err error
	c.closeOnce.Do(func() {
		err = c.conn.Close()
	})
	return err
}

func (c *tcpConn) addKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys[key] = struct{}{}
	if c.peer == "" {
		c.peer = key
	}
}

func (c *tcpConn) setPeer(peer string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.peer = peer
	if peer != "" {
		c.keys[peer] = struct{}{}
	}
}

func (c *tcpConn) keysSnapshot() map[string]struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make(map[string]struct{}, len(c.keys))
	for key := range c.keys {
		keys[key] = struct{}{}
	}
	return keys
}

func (c *tcpConn) markReady() {
	c.once.Do(func() {
		close(c.ready)
	})
}

func (c *tcpConn) failReady(err error) {
	c.mu.Lock()
	if c.handshakeErr == nil {
		c.handshakeErr = err
	}
	c.mu.Unlock()
	c.markReady()
}

func (c *tcpConn) waitReady() error {
	<-c.ready
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.handshakeErr != nil {
		return c.handshakeErr
	}
	return nil
}

func encodeHandshake(addr string) []byte {
	buf := make([]byte, 2+len(addr))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(addr)))
	copy(buf[2:], addr)
	return buf
}

func decodeHandshake(r io.Reader) (string, error) {
	var lengthBuf [2]byte
	if _, err := io.ReadFull(r, lengthBuf[:]); err != nil {
		return "", err
	}
	length := int(binary.BigEndian.Uint16(lengthBuf[:]))
	if length == 0 {
		return "", fmt.Errorf("empty peer address")
	}
	if length > maxHandshakeAddrLen {
		return "", fmt.Errorf("peer address too long: %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}
