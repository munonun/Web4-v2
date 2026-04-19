package transport

import (
	"encoding/binary"
	"fmt"
	"io"

	"web4/node"
	"web4/protocol"
)

const maxFrameSize = 16 << 20

type Handler func(from string, msg protocol.Message) error

type Transport interface {
	Start(handler Handler) error
	Send(peer string, msg protocol.Message) error
	Close() error
}

func EncodeMessageFrame(msg protocol.Message) []byte {
	payload := protocol.EncodeMessage(msg)
	frame := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(payload)))
	copy(frame[4:], payload)
	return frame
}

func DecodeMessageFrame(r io.Reader) (protocol.Message, error) {
	var lengthBuf [4]byte
	if _, err := io.ReadFull(r, lengthBuf[:]); err != nil {
		return protocol.Message{}, err
	}

	frameLen := binary.BigEndian.Uint32(lengthBuf[:])
	if frameLen > maxFrameSize {
		return protocol.Message{}, fmt.Errorf("frame too large: %d", frameLen)
	}

	payload := make([]byte, frameLen)
	if _, err := io.ReadFull(r, payload); err != nil {
		return protocol.Message{}, err
	}

	return protocol.DecodeMessage(payload)
}

func AttachNode(n *node.Node, tr Transport) error {
	n.SetSender(func(to node.PeerID, msg protocol.Message) error {
		return tr.Send(string(to), msg)
	})
	if err := tr.Start(func(from string, msg protocol.Message) error {
		return n.OnMessage(node.PeerID(from), msg)
	}); err != nil {
		n.SetSender(nil)
		return err
	}
	return nil
}
