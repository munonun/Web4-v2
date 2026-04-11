package crypto

type Hash [32]byte

type Hasher interface {
	Sum256(data []byte) [32]byte
}

type Value struct {
	ID     Hash
	Amount uint64
	Unit   string
	Owner  []byte
	Expiry int64
	Depth  int
}

type Transfer struct {
	ID        Hash
	Inputs    []Hash
	Outputs   []Value
	Sig       []byte
	Timestamp int64
}
