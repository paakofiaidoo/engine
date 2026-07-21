package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Bridge manages a persistent JSON-RPC connection to the Node.js worker process.
// It supports concurrent calls by dispatching responses to per-request channels
// keyed by request ID — no global read lock needed.
type Bridge struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdinMu sync.Mutex       // guards stdin writes only
	scanner *bufio.Scanner
	mu      sync.Mutex       // guards nextID and pending map
	nextID  int
	pending map[int]chan Response
}

type Request struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type Response struct {
	Jsonrpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewBridge(workerPath string) (*Bridge, error) {
	cmd := exec.Command("npx", "ts-node", workerPath)
	cmd.Dir = "worker"
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	b := &Bridge{
		cmd:     cmd,
		stdin:   stdin,
		scanner: bufio.NewScanner(stdoutPipe),
		nextID:  1,
		pending: make(map[int]chan Response),
	}

	// Start the async read loop — one goroutine dispatches all responses
	go b.readLoop()

	return b, nil
}

// readLoop continuously reads newline-delimited JSON from the worker stdout
// and dispatches each response to the waiting caller's channel.
func (b *Bridge) readLoop() {
	for b.scanner.Scan() {
		line := b.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			// Non-JSON output (e.g., worker debug logs) — ignore
			fmt.Printf("[bridge] non-JSON from worker: %s\n", string(line))
			continue
		}

		b.mu.Lock()
		ch, ok := b.pending[resp.ID]
		if ok {
			delete(b.pending, resp.ID)
		}
		b.mu.Unlock()

		if ok {
			ch <- resp
		}
	}

	// Worker exited — fail all pending callers
	b.mu.Lock()
	for id, ch := range b.pending {
		ch <- Response{
			ID:    id,
			Error: &RPCError{Code: -32000, Message: "worker process exited"},
		}
	}
	b.pending = make(map[int]chan Response)
	b.mu.Unlock()
}

// Call sends a JSON-RPC request to the worker and waits for the response.
// Concurrent calls are safe — each waits on its own channel keyed by request ID.
func (b *Bridge) Call(method string, params interface{}) (json.RawMessage, error) {
	return b.CallWithTimeout(method, params, 30*time.Second)
}

// CallWithTimeout is like Call but with an explicit timeout.
func (b *Bridge) CallWithTimeout(method string, params interface{}, timeout time.Duration) (json.RawMessage, error) {
	// Register pending channel before sending (avoids race where response arrives first)
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	ch := make(chan Response, 1)
	b.pending[id] = ch
	b.mu.Unlock()

	req := Request{Jsonrpc: "2.0", Method: method, Params: params, ID: id}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	b.stdinMu.Lock()
	_, err = b.stdin.Write(append(reqBytes, '\n'))
	b.stdinMu.Unlock()
	if err != nil {
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
		return nil, fmt.Errorf("failed to write to worker: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("worker error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	case <-ctx.Done():
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
		return nil, fmt.Errorf("worker call timed out after %s (method=%s)", timeout, method)
	}
}

func (b *Bridge) Close() error {
	return b.cmd.Process.Kill()
}
