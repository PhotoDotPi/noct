package player

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"
)

func ipcPair(t *testing.T, responder func(req map[string]any) map[string]any) (*ipcClient, net.Conn) {
	t.Helper()
	clientEnd, serverEnd := net.Pipe()

	go func() {
		dec := json.NewDecoder(serverEnd)
		for {
			var req map[string]any
			if err := dec.Decode(&req); err != nil {
				return
			}
			resp := responder(req)
			id, _ := req["request_id"].(float64)
			resp["request_id"] = int(id)
			b, _ := json.Marshal(resp)
			serverEnd.Write(append(b, '\n'))
		}
	}()

	return newIPC(clientEnd), serverEnd
}

func TestIPCCommandRoundTrip(t *testing.T) {
	c, _ := ipcPair(t, func(req map[string]any) map[string]any {
		return map[string]any{"error": "success", "data": 42.0}
	})
	defer c.Close()

	v, err := c.Command(context.Background(), "get_property", "volume")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 42.0 {
		t.Fatalf("expected data 42.0, got %v", v)
	}
}

func TestIPCCommandError(t *testing.T) {
	c, _ := ipcPair(t, func(req map[string]any) map[string]any {
		return map[string]any{"error": "property unavailable"}
	})
	defer c.Close()

	_, err := c.Command(context.Background(), "get_property", "nope")
	if err == nil {
		t.Fatal("expected an error")
	}
	var ipcErr *IPCError
	if !errors.As(err, &ipcErr) {
		t.Fatalf("expected *IPCError, got %T", err)
	}
}

func TestIPCDeliversEvents(t *testing.T) {
	c, srv := ipcPair(t, func(req map[string]any) map[string]any {
		return map[string]any{"error": "success"}
	})
	defer c.Close()

	evt := map[string]any{"event": "property-change", "id": 1, "data": false, "name": "pause"}
	b, _ := json.Marshal(evt)
	srv.Write(append(b, '\n'))

	select {
	case fr := <-c.Events():
		if fr.Event != "property-change" || fr.ID != 1 {
			t.Fatalf("unexpected event: %+v", fr)
		}
		if ok, _ := fr.DataBool(); ok {
			t.Fatal("expected data false")
		}
	case <-time.After(time.Second):
		t.Fatal("expected an event")
	}
}

func TestIPCMalformedLinesSkipped(t *testing.T) {
	c, srv := ipcPair(t, func(req map[string]any) map[string]any {
		return map[string]any{"error": "success", "data": 7.0}
	})
	defer c.Close()

	srv.Write([]byte("this is not json\n"))
	srv.Write([]byte(`{"error":"success","data":7}` + "\n"))

	v, err := c.Command(context.Background(), "get_property", "volume")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 7.0 {
		t.Fatalf("expected data 7.0, got %v", v)
	}
}

func TestIPCCommandContextCancel(t *testing.T) {
	c, _ := ipcPair(t, func(req map[string]any) map[string]any {
		<-time.After(10 * time.Second)
		return map[string]any{"error": "success"}
	})
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := c.Command(ctx, "get_property", "volume")
	if err == nil {
		t.Fatal("expected a context error")
	}
}

func TestIPCCloseUnblocksPendingCommand(t *testing.T) {
	c, srv := ipcPair(t, func(req map[string]any) map[string]any {
		<-time.After(10 * time.Second)
		return map[string]any{"error": "success"}
	})
	srv.Close()
	start := time.Now()
	_, err := c.Command(context.Background(), "get_property", "volume")
	if err == nil {
		t.Fatal("expected an error after connection closed")
	}
	if time.Since(start) > time.Second {
		t.Fatalf("command waited too long to notice close: %v", time.Since(start))
	}
}

func TestIPCConnectionEOFClosesEvents(t *testing.T) {
	c, srv := ipcPair(t, func(req map[string]any) map[string]any {
		return map[string]any{"error": "success"}
	})
	srv.Close()
	select {
	case _, ok := <-c.Events():
		if ok {
			t.Fatal("expected events channel to be closed, got a frame")
		}
	case <-time.After(time.Second):
		t.Fatal("expected events channel to close after EOF")
	}
}

func TestIPCPendingMapCleanedUp(t *testing.T) {
	c, _ := ipcPair(t, func(req map[string]any) map[string]any {
		return map[string]any{"error": "success", "data": 1.0}
	})
	defer c.Close()
	_, err := c.Command(context.Background(), "get_property", "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c.mu.Lock()
	n := len(c.pending)
	c.mu.Unlock()
	if n != 0 {
		t.Fatalf("expected no pending after command, got %d", n)
	}
}
