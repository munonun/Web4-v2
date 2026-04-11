package crypto

import "crypto/ed25519"

// SignTransfer is a low-level helper for signing an already canonical payload.
func SignTransfer(priv ed25519.PrivateKey, msg []byte) []byte {
	return ed25519.Sign(priv, msg)
}

// VerifyTransfer is a low-level helper for verifying an already canonical payload.
func VerifyTransfer(pub ed25519.PublicKey, msg, sig []byte) bool {
	return ed25519.Verify(pub, msg, sig)
}

func SignCanonicalTransfer(priv ed25519.PrivateKey, t Transfer) []byte {
	return ed25519.Sign(priv, EncodeTransfer(t))
}

func VerifyCanonicalTransfer(pub ed25519.PublicKey, t Transfer, sig []byte) bool {
	return ed25519.Verify(pub, EncodeTransfer(t), sig)
}
