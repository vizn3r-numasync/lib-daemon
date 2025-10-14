package dftp_test

import (
	"fmt"
	"testing"
	"time"

	_ "embed"

	"github.com/vizn3r-numasync/lib-numa/dftp"
)

var m *dftp.Receiver
var ready = make(chan struct{})

func init() {
	dftp.RegisterPacketHandler(dftp.MSG_PIGN, func(p *dftp.Packet, c *dftp.Connection) (*dftp.Packet, error) {
		p.Type = dftp.MSG_POGN // Change type to response
		return p, nil
	})

	// Client-side handler for response (no reply needed)
	dftp.RegisterPacketHandler(dftp.MSG_POGN, func(p *dftp.Packet, c *dftp.Connection) (*dftp.Packet, error) {
		return nil, nil // Don't reply to responses
	})
	m = dftp.NewReceiver("127.0.0.1", 3387)
	go func() {
		if err := m.Listen(ready); err != nil {
			panic(err)
		}
	}()

	<-ready
}

//go:embed test.txt
var text []byte

func TestTransferInitSmall(b *testing.T) {
	s := dftp.NewSession("127.0.0.1", 3387, 0, 1)

	fmt.Println("Sending...")
	s.Send([]byte("Hello world!"))
	fmt.Println("Receiving...")
	s.Recv()
	fmt.Println("Done")
}

func TestTransferInitBig(b *testing.T) {
	s := dftp.NewSession("127.0.0.1", 3387, 0, 1)

	fmt.Println("Sending...")
	s.Send(text)
	fmt.Println("Receiving...")
	s.Recv()
	fmt.Println("Done")
}

func BenchmarkSingleRequest(b *testing.B) {
	time.Sleep(time.Millisecond * 100)
	p := &dftp.Packet{
		Type: dftp.MSG_PIGN,
		Data: []byte("PIGN!"),
	}

	b.ResetTimer()
	for b.Loop() {
		conn, _ := dftp.NewConn("127.0.0.1", 3387, 0)
		conn.Send(p)
		conn.Recv()
		conn.Close()
	}
	m.CloseAllConnections()
}

func BenchmarkConcurrent(b *testing.B) {
	time.Sleep(time.Millisecond * 100)
	p := &dftp.Packet{
		Type: dftp.MSG_PIGN,
		Data: []byte("PIGN!"),
	}

	//var mu sync.Mutex
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, _ := dftp.NewConn("127.0.0.1", 3387, 0)
			conn.Send(p)
			conn.Recv()
			conn.Close()
		}
	})
	m.CloseAllConnections()
}

func BenchmarkWithPool(b *testing.B) {
	time.Sleep(time.Millisecond * 100)
	p := &dftp.Packet{
		Type: dftp.MSG_PIGN,
		Data: []byte("PIGN!"),
	}

	// Pre-create connection pool
	poolSize := 10
	conns := make([]*dftp.Connection, poolSize)
	for i := range conns {
		conns[i], _ = dftp.NewConn("127.0.0.1", 3387, 0)
	}
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		conn := conns[i%poolSize]
		conn.Send(p)
		conn.Recv()
	}
	m.CloseAllConnections()
}

// BenchmarkSingleRequest1400B benchmarks a single packet of 1400 bytes.

func BenchmarkSingleRequest1400B(b *testing.B) {
	time.Sleep(time.Millisecond * 100)
	p := &dftp.Packet{
		Type: dftp.MSG_PIGN,
		Data: make([]byte, 1400), // Full packet
	}

	b.ResetTimer()
	for b.Loop() {
		conn, _ := dftp.NewConn("127.0.0.1", 3387, 0)
		conn.Send(p)
		conn.Recv()
		conn.Close()
	}
	m.CloseAllConnections()
}

func BenchmarkConcurrent1400B(b *testing.B) {
	time.Sleep(time.Millisecond * 100)
	p := &dftp.Packet{
		Type: dftp.MSG_PIGN,
		Data: make([]byte, 1400),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, _ := dftp.NewConn("127.0.0.1", 3387, 0)
			conn.Send(p)
			conn.Recv()
			conn.Close()
		}
	})
	m.CloseAllConnections()
}

func BenchmarkWithPool1400B(b *testing.B) {
	time.Sleep(time.Millisecond * 100)
	p := &dftp.Packet{
		Type: dftp.MSG_PIGN,
		Data: make([]byte, 1400),
	}

	poolSize := 10
	conns := make([]*dftp.Connection, poolSize)
	for i := range conns {
		conns[i], _ = dftp.NewConn("127.0.0.1", 3387, 0)
	}
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		conn := conns[i%poolSize]
		conn.Send(p)
		conn.Recv()
	}
	m.CloseAllConnections()
}
