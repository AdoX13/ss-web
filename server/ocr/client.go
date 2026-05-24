package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Client is what the API uses to talk to the sandboxed OCR worker.
type Client interface {
	Process(ctx context.Context, documentID string, image []byte) (*Result, error)
	Close() error
}

// MaxRequestBytes is the hard cap enforced client-side, matching the
// worker's boundary cap. Sending more is a bug the worker will reject
// with 413 — we surface that earlier as a clean error.
const MaxRequestBytes = 10 * 1024 * 1024 // 10 MB

// DefaultRequestTimeout caps a single Process() call end-to-end. It is
// slightly larger than the worker's internal 30 s budget to leave room
// for IPC overhead.
const DefaultRequestTimeout = 35 * time.Second

// UnixSocketClient is the production implementation: HTTP over a Unix
// domain socket to the gVisor-sandboxed worker.
type UnixSocketClient struct {
	socketPath string
	http       *http.Client
}

// NewUnixSocketClient builds a Client that dials the given Unix socket.
// The socket file must be writable by the API's UID (Docker volume
// permissions are configured to ensure this).
func NewUnixSocketClient(socketPath string) *UnixSocketClient {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
		// We talk to a single peer over a Unix socket; large pool sizes
		// are pointless and pin file descriptors.
		MaxIdleConns:        2,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     30 * time.Second,
		// No proxy, no TLS — this is a local Unix socket.
	}
	return &UnixSocketClient{
		socketPath: socketPath,
		http: &http.Client{
			Transport: transport,
			Timeout:   DefaultRequestTimeout,
		},
	}
}

// Process sends an image to the worker and returns a validated Result.
//
// It does not retry: OCR is idempotent in principle but expensive in
// practice, and a retried request that hits the same crash bug just
// doubles the load. The caller can implement retry policy if needed.
func (c *UnixSocketClient) Process(ctx context.Context, documentID string, image []byte) (*Result, error) {
	if documentID == "" {
		return nil, errors.New("ocr: document_id is required")
	}
	if len(image) == 0 {
		return nil, errors.New("ocr: empty image payload")
	}
	if len(image) > MaxRequestBytes {
		return nil, fmt.Errorf("ocr: image %d bytes exceeds %d byte cap", len(image), MaxRequestBytes)
	}

	// The Host header is irrelevant for a Unix socket dial but the Go
	// HTTP client requires a valid URL. Use a placeholder.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://ocr-worker/process", bytes.NewReader(image))
	if err != nil {
		return nil, fmt.Errorf("ocr: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Document-Id", documentID)
	req.ContentLength = int64(len(image))

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ocr: dial worker: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("ocr: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp.StatusCode, body)
	}

	result, err := DecodeResult(body)
	if err != nil {
		return nil, err
	}
	if result.DocumentID != documentID {
		return nil, fmt.Errorf("ocr: document_id mismatch: requested %q, got %q",
			documentID, result.DocumentID)
	}
	return result, nil
}

// Close releases idle connections held by the underlying transport.
func (c *UnixSocketClient) Close() error {
	c.http.CloseIdleConnections()
	return nil
}

func parseError(status int, body []byte) error {
	var er ErrorResponse
	if err := json.Unmarshal(body, &er); err == nil && er.Error != "" {
		return &WorkerError{Status: status, Code: er.Code, Message: er.Error}
	}
	return &WorkerError{Status: status, Message: string(body)}
}

// WorkerError is returned for non-2xx responses from the worker.
// Callers can type-assert to inspect the HTTP status and error code.
type WorkerError struct {
	Status  int
	Code    string
	Message string
}

func (e *WorkerError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("ocr worker: %d %s: %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("ocr worker: %d: %s", e.Status, e.Message)
}

// bytesReader is a tiny shim so schema.go can avoid importing "bytes"
// directly (keeps the schema file's stdlib surface minimal).
func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }
