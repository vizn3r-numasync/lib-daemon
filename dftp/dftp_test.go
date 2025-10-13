package dftp_test

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/vizn3r-numasync/lib-numa/dftp"
)

func TestSimplePingPong(t *testing.T) {
	str := os.Getenv("N")
	n, _ := strconv.Atoi(str)
	p := &dftp.Packet{
		Type: dftp.MSG_PIGN,
		Data: []byte("PIGN!"),
	}

	var wg sync.WaitGroup

	start := time.Now()
	for range n {
		wg.Go(func() {
			conn, err := dftp.NewConn("127.0.0.1", 338, 0)
			if err != nil {
				t.Fatal(err)
				return
			}
			defer conn.Close()
			conn.Send(p)
			packet, err := conn.Recv()
			if err != nil {
				t.Fatal(err)
				return
			}
			if string(packet.Data) != "POGN!" {
				t.Fatal("Wrong data")
			}
		})
	}
	wg.Wait()
	m.Close()
	fmt.Println("Time taken", time.Since(start).Microseconds())

}
