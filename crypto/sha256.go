package crypto

import "crypto/sha256"

type SHA256Hasher struct{}

func (SHA256Hasher) Sum256(data []byte) [32]byte {
	return sha256.Sum256(data)
}
