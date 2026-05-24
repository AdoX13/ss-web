//go:build !cgo

package pipeline

import "errors"

// TesseractPool stub for CGO-disabled builds (lint, static analysis,
// non-Docker test runs). The real implementation lives in
// engine_tesseract.go and requires CGO + libtesseract.
type TesseractPool struct{}

// NewTesseractPool returns a clear error rather than a nil pool that
// would panic on first use.
func NewTesseractPool(_ int, _ []string) (*TesseractPool, error) {
	return nil, errors.New("ocr-worker built without CGO; Tesseract unavailable")
}

func (*TesseractPool) Recognize(_ []byte) (*OCROutput, error) {
	return nil, errors.New("ocr-worker built without CGO")
}

func (*TesseractPool) Close() error { return nil }
