package dftp_test

import (
	"github.com/vizn3r-numasync/lib-numa/dftp"
	"testing"
)

func TestConnection_CreateStreams(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		ip   string
		port int
		// Named input parameters for target function.
		numStreams uint32
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := dftp.NewConn(tt.ip, tt.port)
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			gotErr := conn.CreateStreams(tt.numStreams)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("CreateStreams() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("CreateStreams() succeeded unexpectedly")
			}
		})
	}
}
