package crypto

func ComputeValueID(v Value) Hash {
	sum := Blake3Hasher{}.Sum256(EncodeValue(v))
	return Hash(sum)
}

func ComputeTransferID(t Transfer) Hash {
	sum := Blake3Hasher{}.Sum256(EncodeTransfer(t))
	return Hash(sum)
}
