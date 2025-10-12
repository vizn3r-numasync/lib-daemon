package dftp

import (
	"bytes"
	"encoding/binary"
)

type Chunk struct {
	ID       uint32
	Checksum uint32
	Data     []byte

	received bool
}

type ChunkTransfer struct {
	ID     uint32
	Chunks []*Chunk
}

func NewChunkMap() map[uint32]*Chunk {
	return make(map[uint32]*Chunk)
}

type Chunks []*Chunk

func NewChunkIDMap() map[uint32]*Chunk {
	return make(map[uint32]*Chunk)
}

func NewChunkStreamMap() map[uint8]*Chunks {
	return make(map[uint8]*Chunks)
}

func ChunkIDMaptoStreamMap(chunkIDMap map[uint32]*Chunk, streamNum uint8) map[uint8]*Chunks {
	chunkMaps := make(map[uint8]*Chunks)
	for chunkID, chunk := range chunkIDMap {
		connID := uint8(chunkID) % streamNum
		if _, ok := chunkMaps[connID]; !ok {
			chunkMaps[connID] = &Chunks{}
		}
		*chunkMaps[connID] = append(*chunkMaps[connID], chunk)
	}
	return chunkMaps
}

func (c *Chunks) Serialize() []byte {
	chunkMapData := bytes.NewBuffer([]byte{})
	for _, chunk := range *c {
		binary.Write(chunkMapData, binary.LittleEndian, chunk.ID)
		chunkMapData.WriteString(":")
		binary.Write(chunkMapData, binary.LittleEndian, chunk.Checksum)
		chunkMapData.WriteString(";")
	}
	return chunkMapData.Bytes()
}

func (c *Chunks) Deserialize(data []byte) {
	chunks := bytes.Split(data, []byte(";"))
	for _, chunk := range chunks {
		chunkID := binary.LittleEndian.Uint32(chunk[:4])
		checksum := binary.LittleEndian.Uint32(chunk[4:8])
		data := chunk[8:]
		c.AddChunk(&Chunk{
			ID:       chunkID,
			Checksum: checksum,
			Data:     data,
		})
	}
}

func (c *Chunks) AddChunk(chunk *Chunk) {
	if _, ok := c.Get(chunk.ID); !ok {
		*c = append(*c, chunk)
	}
}

func (c *Chunks) Get(chunkID uint32) (*Chunk, bool) {
	for _, chunk := range *c {
		if chunk.ID == chunkID {
			return chunk, true
		}
	}
	return nil, false
}
