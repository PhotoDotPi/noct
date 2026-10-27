//go:build !windows

package player

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

func ipcEndpoint() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("noct-mpv-%d.sock", os.Getpid()))
}

func dialIPC(ctx context.Context, endpoint string) (net.Conn, error) {
	for {
		conn, err := net.DialTimeout("unix", endpoint, 300*time.Millisecond)
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
