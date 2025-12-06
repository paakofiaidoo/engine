package bridge

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

type Bridge struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mutex  sync.Mutex
	nextID int
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
	Error   *Error          `json:"error,omitempty"`
	ID      int             `json:"id"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewBridge(workerPath string) (*Bridge, error) {
	// Assuming ts-node is available or we run the compiled js
	// For dev, let's try npx ts-node
	cmd := exec.Command("npx", "ts-node", workerPath)

	// Set working directory to where package.json is
	cmd.Dir = "worker"
	// Adjust this path logic to be more robust later

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// Capture stderr for debugging
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &Bridge{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdoutPipe),
		nextID: 1,
	}, nil
}

func (b *Bridge) Call(method string, params interface{}) (json.RawMessage, error) {
	b.mutex.Lock()
	id := b.nextID
	b.nextID++
	b.mutex.Unlock()

	req := Request{
		Jsonrpc: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Send request
	b.mutex.Lock() // Lock for writing to stdin
	_, err = b.stdin.Write(append(reqBytes, '\n'))
	b.mutex.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to write to worker: %w", err)
	}

	// Read response (Blocking for now, naive implementation)
	// In a real scenario, we'd have a loop reading stdout and dispatching to channels based on ID
	// For this MVP, we assume synchronous 1-1 request/response for simplicity,
	// BUT since we share the bridge, we need to be careful.
	// A better approach is a dedicated read loop.
	// Let's stick to a simple mutex-protected read/write for now, assuming low concurrency or sequential usage.

	b.mutex.Lock()
	defer b.mutex.Unlock()

	if !b.stdout.Scan() {
		return nil, fmt.Errorf("worker closed connection or failed to scan: %v", b.stdout.Err())
	}

	respBytes := b.stdout.Bytes()
	var resp Response
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse worker response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("worker error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}

func (b *Bridge) Close() error {
	return b.cmd.Process.Kill()
}
