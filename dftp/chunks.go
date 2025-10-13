package dftp

import (
	"bytes"
	"encoding/binary"
	"sync"
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

type Chunks struct {
	chunks []*Chunk
	mu     sync.RWMutex
}

func NewChunk() *Chunk {
	return &Chunk{
		ID:       0,
		Checksum: 0,
		Data:     nil,
		received: false,
	}
}

// Chunks mapped to a chunkID
func NewChunkMap() map[uint32]*Chunk {
	return make(map[uint32]*Chunk)
}

// Chunk arrays mapped to a steamID
func NewChunksMap() map[uint8]*Chunks {
	return make(map[uint8]*Chunks)
}

// NewChunks creates a new Chunks object
func NewChunks() *Chunks {
	return &Chunks{
		chunks: make([]*Chunk, 0),
	}
}

func ChunkIDMaptoStreamMap(chunkIDMap map[uint32]*Chunk, streamNum uint8) map[uint8]*Chunks {
	chunkMaps := make(map[uint8]*Chunks)
	for chunkID, chunk := range chunkIDMap {
		connID := uint8(chunkID) % streamNum
		if _, ok := chunkMaps[connID]; !ok {
			chunkMaps[connID] = &Chunks{}
		}
		chunkMaps[connID].AddChunk(chunk)
	}
	return chunkMaps
}

func (c *Chunks) SerializeMap() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	buf := bytes.NewBuffer(nil)

	binary.Write(buf, binary.LittleEndian, uint32(len(c.chunks)))

	for _, chunk := range c.chunks {
		binary.Write(buf, binary.LittleEndian, chunk.ID)
		binary.Write(buf, binary.LittleEndian, chunk.Checksum)
	}
	return buf.Bytes()
}

func (c *Chunks) DeserializeMap(data []byte) error {
	buf := bytes.NewBuffer(data)

	var numChunks uint32
	if err := binary.Read(buf, binary.LittleEndian, &numChunks); err != nil {
		return err
	}

	for range numChunks {
		var chunkID uint32
		var checksum uint32
		if err := binary.Read(buf, binary.LittleEndian, &chunkID); err != nil {
			return err
		}
		if err := binary.Read(buf, binary.LittleEndian, &checksum); err != nil {
			return err
		}
		c.AddChunk(&Chunk{
			ID:       chunkID,
			Checksum: checksum,
			Data:     nil,
			received: false,
		})
	}
	return nil
}

// chunkID - checksum map
func (c *Chunks) ChunkMap() map[uint32]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	chunkMap := make(map[uint32]string)
	for _, chunk := range c.chunks {
		chunkMap[chunk.ID] = string(chunk.Checksum)
	}
	return chunkMap
}

func (c *Chunks) AddChunk(chunk *Chunk) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.Get(chunk.ID); !ok {
		c.chunks = append(c.chunks, chunk)
	}
}

func (c *Chunks) Get(chunkID uint32) (*Chunk, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, chunk := range c.chunks {
		if chunk.ID == chunkID {
			return chunk, true
		}
	}
	return nil, false
}

func (c *Chunks) Remove(chunkID uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, chunk := range c.chunks {
		if chunk.ID == chunkID {
			c.chunks = append(c.chunks[:i], c.chunks[i+1:]...)
			return
		}
	}
}

func (c *Chunks) Next() *Chunk {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, chunk := range c.chunks {
		if !chunk.received {
			chunk.received = true
			return chunk
		}
	}
	return nil
}
