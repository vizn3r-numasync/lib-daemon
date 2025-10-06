package dftp_test

import (
	"log"
	"testing"
	"time"

	"github.com/vizn3r-numasync/lib-numa/dftp"
)

func TestPingPong(t *testing.T) {
	go func() {
		if err := dftp.UDPListen("127.0.0.1", 3387); err != nil {
			log.Println(err)
		}
	}()

	p := &dftp.Packet{
		Flags: dftp.FLAG_SYN,
		Type:  dftp.MSG_CONN_PING,
		Data:  []byte("PIGN!"),
	}
	for range 10 {
		conn, err := dftp.Dial("127.0.0.1", 3387)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		conn.Send(p)
		time.Sleep(time.Millisecond * 100)
		p, err := conn.Receive(dftp.MSG_CONN_PONG, time.Second*1)
		if err != nil {
			t.Fatal(err)
		}
		if string(p.Data) != "POGN!" {
			t.Fatal("unexpected data")
		}

	}
}
