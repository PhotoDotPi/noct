package player

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

type ipcClient struct {
	conn net.Conn

	writeMu sync.Mutex
	mu      sync.Mutex
	nextID  int
	pending map[int]chan ipcFrame

	events  chan ipcFrame
	closeCh chan struct{}
	once    sync.Once
}

type ipcFrame struct {
	Event     string          `json:"event"`
	RequestID int             `json:"request_id"`
	Error     string          `json:"error"`
	Data      json.RawMessage `json:"data"`
	Name      string          `json:"name"`
	ID        int             `json:"id"`
	Reason    string          `json:"reason"`
}

func (f ipcFrame) DataBool() (bool, bool) {
	var b bool
	if len(f.Data) == 0 || json.Unmarshal(f.Data, &b) != nil {
		return false, false
	}
	return b, true
}

func (f ipcFrame) DataFloat() (float64, bool) {
	var v float64
	if len(f.Data) == 0 || json.Unmarshal(f.Data, &v) != nil {
		return 0, false
	}
	return v, true
}

func newIPC(conn net.Conn) *ipcClient {
	c := &ipcClient{
		conn:    conn,
		pending: make(map[int]chan ipcFrame),
		events:  make(chan ipcFrame, 64),
		closeCh: make(chan struct{}),
	}
	go c.run()
	return c
}

func (c *ipcClient) Command(ctx context.Context, args ...any) (any, error) {
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan ipcFrame, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	req := map[string]any{"command": args, "request_id": id}
	line, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	c.writeMu.Lock()
	_, werr := c.conn.Write(append(line, '\n'))
	c.writeMu.Unlock()
	if werr != nil {
		return nil, werr
	}

	select {
	case fr := <-ch:
		if fr.Error != "" && fr.Error != "success" {
			return nil, &IPCError{Command: fmt.Sprint(args[0]), Message: fr.Error}
		}
		if len(fr.Data) == 0 {
			return nil, nil
		}
		var v any
		if err := json.Unmarshal(fr.Data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closeCh:
		return nil, fmt.Errorf("ipc closed")
	}
}

func (c *ipcClient) Events() <-chan ipcFrame {
	return c.events
}

func (c *ipcClient) Close() error {
	c.once.Do(func() {
		close(c.closeCh)
		c.conn.Close()
	})
	return nil
}

func (c *ipcClient) run() {
	defer close(c.events)
	sc := bufio.NewScanner(c.conn)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var fr ipcFrame
		if err := json.Unmarshal(line, &fr); err != nil {
			continue
		}

		if fr.Event != "" {
			select {
			case c.events <- fr:
			case <-c.closeCh:
				return
			}
			continue
		}

		c.mu.Lock()
		ch := c.pending[fr.RequestID]
		c.mu.Unlock()
		if ch != nil {
			select {
			case ch <- fr:
			default:
			}
		}
	}
	if err := sc.Err(); err != nil {
		return
	}
}
