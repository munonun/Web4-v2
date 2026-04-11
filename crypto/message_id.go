package crypto

import (
	"crypto/rand"
	"io"
)

func GenerateMessageID() [16]byte {
	var id [16]byte
	if _, err := io.ReadFull(rand.Reader, id[:]); err != nil {
		panic(err)
	}
	return id
}
