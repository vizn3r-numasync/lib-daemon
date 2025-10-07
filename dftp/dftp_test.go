package dftp_test

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/vizn3r-numasync/lib-numa/dftp"
)

func TestPingPong(t *testing.T) {
	go func() {
		conn := dftp.NewConnManager("127.0.0.1", 3387)
		if err := conn.Listen(); err != nil {
			log.Println(err)
		}
	}()

	time.Sleep(time.Millisecond * 500)

	p := &dftp.Packet{
		Flags: dftp.FLAG_SYN,
		Type:  dftp.MSG_CONN_PING,
		Data:  []byte("PIGN!"),
	}
	for i := range 5 {
		fmt.Println("\nSending PING", i)
		fmt.Println("------------------------------------------")
		conn, err := dftp.NewConn("127.0.0.1", 3387)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		conn.Send(p)
		resp, err := conn.Receive(dftp.MSG_CONN_PONG, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if string(resp.Data) != "POGN!" {
			t.Fatal("unexpected data")
		}

	}
}

func TestMultiplePingPong(t *testing.T) {
	go func() {
		conn := dftp.NewConnManager("127.0.0.1", 3388)
		if err := conn.Listen(); err != nil {
			log.Println(err)
		}
	}()

	time.Sleep(time.Millisecond * 500)

	p := &dftp.Packet{
		Flags: dftp.FLAG_SYN,
		Type:  dftp.MSG_CONN_PING,
		Data:  []byte("PIGN!"),
	}
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			conn, err := dftp.NewConn("127.0.0.1", 3387)
			if err != nil {
				log.Println(err)
				return
			}
			defer conn.Close()
			conn.Send(p)
			resp, err := conn.Receive(dftp.MSG_CONN_PONG, time.Second)
			if err != nil {
				log.Println(err)
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
