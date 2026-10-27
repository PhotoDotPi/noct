//go:build windows

package player

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	npipe "gopkg.in/natefinch/npipe.v2"
)

func ipcEndpoint() string {
	return fmt.Sprintf(`\\.\pipe\noct-mpv-%d`, os.Getpid())
}

func dialIPC(ctx context.Context, endpoint string) (net.Conn, error) {
	for {
		conn, err := npipe.DialTimeout(endpoint, 300*time.Millisecond)
		if err == nil {
			return conn, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
}
