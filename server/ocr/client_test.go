package ocr

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeWorker serves OCR responses over a Unix socket — same wire shape
// the real worker will use. It lets us exercise the full client path
// without bringing up Docker / gVisor / Tesseract.
type fakeWorker struct {
	socketPath string
	server     *http.Server
	listener   net.Listener
	handler    http.HandlerFunc
}

func startFakeWorker(t *testing.T, handler http.HandlerFunc) *fakeWorker {
	t.Helper()
	// macOS caps Unix socket paths at ~104 chars (sun_path), which
	// t.TempDir() blows through on /var/folders/.... Put the socket
	// in /tmp directly with a short name.
	dir, err := os.MkdirTemp("/tmp", "ocrt")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socketPath := filepath.Join(dir, "s")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	return &fakeWorker{socketPath: socketPath, server: srv, listener: ln, handler: handler}
}

func (f *fakeWorker) Close() { _ = f.server.Close(); _ = os.Remove(f.socketPath) }

func TestUnixSocketClient_Process_HappyPath(t *testing.T) {
	want := validResult()
	want.DocumentID = "doc-42"
	f := startFakeWorker(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/process" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-Document-Id"); got != "doc-42" {
			t.Errorf("X-Document-Id: want doc-42 got %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "imagebytes" {
			t.Errorf("body: want imagebytes got %q", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	})
	defer f.Close()

	c := NewUnixSocketClient(f.socketPath)
	defer c.Close()

	got, err := c.Process(context.Background(), "doc-42", []byte("imagebytes"))
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if got.DocumentID != "doc-42" {
		t.Errorf("doc_id: want doc-42 got %q", got.DocumentID)
	}
	if got.OverallConfidence != want.OverallConfidence {
		t.Errorf("confidence drift: want %v got %v", want.OverallConfidence, got.OverallConfidence)
	}
}

func TestUnixSocketClient_Process_DocumentIDMismatch(t *testing.T) {
	bad := validResult()
	bad.DocumentID = "different-id"
	f := startFakeWorker(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(bad)
	})
	defer f.Close()

	c := NewUnixSocketClient(f.socketPath)
	defer c.Close()

	_, err := c.Process(context.Background(), "expected-id", []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "document_id mismatch") {
		t.Fatalf("expected document_id mismatch error, got: %v", err)
	}
}

func TestUnixSocketClient_Process_WorkerErrorIsTyped(t *testing.T) {
	f := startFakeWorker(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Error: "no fields extracted", Code: "no_fields"})
	})
	defer f.Close()

	c := NewUnixSocketClient(f.socketPath)
	defer c.Close()

	_, err := c.Process(context.Background(), "doc-1", []byte("x"))
	we, ok := err.(*WorkerError)
	if !ok {
		t.Fatalf("expected *WorkerError, got %T: %v", err, err)
	}
	if we.Status != http.StatusUnprocessableEntity {
		t.Errorf("status: want 422 got %d", we.Status)
	}
	if we.Code != "no_fields" {
		t.Errorf("code: want no_fields got %q", we.Code)
	}
}

func TestUnixSocketClient_Process_RejectsEmptyImage(t *testing.T) {
	c := NewUnixSocketClient("/nonexistent")
	defer c.Close()
	_, err := c.Process(context.Background(), "doc", nil)
	if err == nil {
		t.Fatal("expected error for empty image")
	}
}

func TestUnixSocketClient_Process_RejectsEmptyDocID(t *testing.T) {
	c := NewUnixSocketClient("/nonexistent")
	defer c.Close()
	_, err := c.Process(context.Background(), "", []byte("x"))
	if err == nil {
		t.Fatal("expected error for empty document_id")
	}
}

func TestUnixSocketClient_Process_RejectsOversizeImage(t *testing.T) {
	c := NewUnixSocketClient("/nonexistent")
	defer c.Close()
	big := make([]byte, MaxRequestBytes+1)
	_, err := c.Process(context.Background(), "doc", big)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected size-cap error, got: %v", err)
	}
}
