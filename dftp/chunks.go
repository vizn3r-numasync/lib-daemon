package dftp

type Chunk struct {
	ID       uint32
	Checksum uint32
	Data     []byte

	Received bool
}

type ChunkTransfer struct {
	ID     uint32
	Chunks []*Chunk
}

func NewChunk() *Chunk {
	return &Chunk{
		ID:       0,
		Checksum: 0,
		Data:     nil,
		Received: false,
	}
}
