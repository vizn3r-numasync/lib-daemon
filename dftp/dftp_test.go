package dftp_test

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/vizn3r-numasync/lib-numa/dftp"
)

var ready = make(chan struct{})

func init() {
	go func() {
		conn := dftp.NewConnManager("127.0.0.1", 3387)
		ready <- struct{}{}
		if err := conn.Listen(); err != nil {
			log.Println(err)
		}
	}()
}

func TestPingPong(t *testing.T) {
	<-ready
	p := &dftp.Packet{
		Flags: dftp.FLAG_SYN,
		Type:  dftp.MSG_PIGN,
		Data:  []byte("PIGN!"),
	}
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			conn, err := dftp.NewConn("127.0.0.1", 3387)
			if err != nil {
				t.Fatal(err)
				return
			}
			defer conn.Close()
			conn.Send(p)
			resp, err := conn.Receive(dftp.MSG_POGN, time.Second)
			if err != nil {
				t.Fatal(err)
				return
			}
			if string(resp.Data) != "POGN!" {
				log.Println("unexpected data")
				return
			}
		})
	}
	wg.Wait()
}
