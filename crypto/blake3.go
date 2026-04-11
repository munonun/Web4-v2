package crypto

import "github.com/zeebo/blake3"

type Blake3Hasher struct{}

func (Blake3Hasher) Sum256(data []byte) [32]byte {
	return blake3.Sum256(data)
}
