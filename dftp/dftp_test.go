package dftp_test

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/vizn3r-numasync/lib-numa/dftp"
)

var ready = make(chan struct{})
var m *dftp.ConnManager

func init() {
	go func() {
		m = dftp.NewConnManager("127.0.0.1", 3387)
		if err := m.Listen(ready); err != nil {
			log.Println(err)
		}
	}()
}

func TestPingPong(t *testing.T) {
	str := os.Getenv("N")
	n, _ := strconv.Atoi(str)
	<-ready
	p := &dftp.Packet{
		Flags: dftp.FLAG_SYN,
		Type:  dftp.MSG_PIGN,
		Data:  []byte("PIGN!"),
	}
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			fmt.Println(i, "Waiting for connection")
			conn, err := dftp.NewConn("127.0.0.1", 3387)
			if err != nil {
				t.Fatal(err)
				return
			}
			defer conn.Close()
			fmt.Println(i, "Sending packet")
			conn.Send(p)
			time.Sleep(10 * time.Millisecond) // Give time for response
			fmt.Println(i, "Waiting for response")
			packet, err := conn.Recv()
			if err != nil {
				t.Fatal(err)
				return
			}
			if string(packet.Data) != "POGN!" {
				t.Fatal("Wrong data")
			}
			fmt.Println(i, "Closing connection")
		})
	}
	wg.Wait()
	m.Close()
}
