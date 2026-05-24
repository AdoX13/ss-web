// Command ocr-worker serves OCR requests over a Unix socket.
//
// It is designed to run as a separate container under gVisor with no
// network and a read-only root filesystem. See docs/architecture/ocr_sandbox.md.
//
// Wire protocol: POST /process with raw JPEG/PNG bytes; returns an
// OcrResult JSON document (team plan v3 §6.2).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"medsec-ocr/ocr-worker/confidence"
	"medsec-ocr/ocr-worker/pipeline"
)

const (
	// MaxRequestBytes mirrors the API client cap (10 MB). Anything
	// larger is rejected with 413 before any decode.
	MaxRequestBytes = 10 * 1024 * 1024

	// RequestTimeout is the end-to-end budget for a single image.
	// Preprocess + OCR + extract all share it.
	RequestTimeout = 30 * time.Second

	// EngineVersion is reported in OcrResult.engine.version. It is set
	// at build time via -ldflags "-X main.EngineVersion=...". A package
	// var (not a const) keeps the linker happy.
	defaultEngineVersion = "tesseract-5.x"
)

// EngineVersion is overridden by ldflags in the Dockerfile.
var EngineVersion = defaultEngineVersion

func main() {
	socketPath := flag.String("socket", "/run/ocr/ocr.sock", "Unix socket path")
	poolSize := flag.Int("pool", 4, "Number of Tesseract client instances in the pool")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	tess, err := pipeline.NewTesseractPool(*poolSize, []string{"eng", "ron"})
	if err != nil {
		logger.Error("init tesseract pool", "err", err)
		os.Exit(1)
	}
	defer tess.Close()

	srv := &workerServer{
		engineVersion: EngineVersion,
		ocr:           tess,
		log:           logger,
		// Limit concurrent in-flight requests to the pool size. Extra
		// callers get 503 instead of stacking up against the Unix socket.
		sem: make(chan struct{}, *poolSize),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/process", srv.handleProcess)
	mux.HandleFunc("/healthz", srv.handleHealth)

	httpSrv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       RequestTimeout,
		WriteTimeout:      RequestTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	listener, err := listenUnix(*socketPath)
	if err != nil {
		logger.Error("listen unix", "path", *socketPath, "err", err)
		os.Exit(1)
	}
	defer listener.Close()
	defer os.Remove(*socketPath)

	logger.Info("ocr-worker listening", "socket", *socketPath, "pool", *poolSize)

	// Graceful shutdown on SIGTERM/SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}

// listenUnix opens a Unix socket with the parent directory ensured and
// 0660 file permissions so only the worker UID and the configured GID
// (shared with the API container) can connect.
func listenUnix(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("mkdir socket dir: %w", err)
	}
	_ = os.Remove(path) // stale socket from previous run
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o660); err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("chmod socket: %w", err)
	}
	return ln, nil
}

type workerServer struct {
	engineVersion string
	ocr           pipeline.OCREngine
	log           *slog.Logger
	sem           chan struct{}

	mu        sync.Mutex
	inFlight  int
	completed uint64
}

func (s *workerServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "ok",
		"engine":     s.engineVersion,
		"in_flight":  s.inFlight,
		"completed":  s.completed,
		"pool_depth": cap(s.sem),
	})
}

func (s *workerServer) handleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	documentID := r.Header.Get("X-Document-Id")
	if documentID == "" {
		writeErr(w, http.StatusBadRequest, "missing_doc_id", "X-Document-Id header is required")
		return
	}

	// Reject oversize before reading the body. Trust Content-Length here
	// only as a fast-fail; the LimitReader below is the real cap.
	if r.ContentLength > MaxRequestBytes {
		writeErr(w, http.StatusRequestEntityTooLarge, "too_large",
			fmt.Sprintf("body %d > max %d", r.ContentLength, MaxRequestBytes))
		return
	}

	// Acquire a concurrency slot or fail fast with 503.
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	default:
		writeErr(w, http.StatusServiceUnavailable, "busy", "worker at capacity")
		return
	}

	s.mu.Lock()
	s.inFlight++
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.inFlight--
		s.completed++
		s.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	// LimitReader enforces the hard cap even if Content-Length lied.
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBytes+1))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "read_body", err.Error())
		return
	}
	if len(body) == 0 {
		writeErr(w, http.StatusBadRequest, "empty_body", "image bytes required")
		return
	}
	if len(body) > MaxRequestBytes {
		writeErr(w, http.StatusRequestEntityTooLarge, "too_large", "body exceeds max")
		return
	}

	start := time.Now()
	result, err := pipeline.Extract(ctx, pipeline.ExtractInput{
		DocumentID:    documentID,
		ImageBytes:    body,
		Engine:        s.ocr,
		EngineVersion: s.engineVersion,
		Now:           time.Now,
	})
	if err != nil {
		s.log.Warn("extract failed", "doc_id", documentID, "err", err)
		writeExtractError(w, err)
		return
	}
	result.ProcessingMs = int(time.Since(start) / time.Millisecond)

	// Compute overall confidence + review flag at the very end so the
	// numbers reflect what's actually in the result.
	result.OverallConfidence, result.NeedsReview = confidence.Compute(result.Fields)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		// Response already started; just log.
		s.log.Error("encode response", "err", err)
	}
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
		"code":  code,
	})
}

func writeExtractError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pipeline.ErrUnsupportedMIME):
		writeErr(w, http.StatusBadRequest, "unsupported_mime", err.Error())
	case errors.Is(err, pipeline.ErrImageTooLarge):
		writeErr(w, http.StatusBadRequest, "image_too_large", err.Error())
	case errors.Is(err, pipeline.ErrImageDecode):
		writeErr(w, http.StatusBadRequest, "image_decode", err.Error())
	case errors.Is(err, pipeline.ErrNoFieldsExtracted):
		writeErr(w, http.StatusUnprocessableEntity, "no_fields", err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
	}
}
