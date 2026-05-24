package ocr

import (
	"encoding/json"
	"testing"
)

// FuzzDecodeResult feeds arbitrary bytes into DecodeResult.
// Invariant: it must never panic, never return a *Result that fails
// Validate(), and must return an error for any input that isn't a
// valid OcrResult JSON document.
//
// Required by Lab 10 Ex 1 (`go test -fuzz`). Run with:
//
//	go test -fuzz=FuzzDecodeResult -fuzztime=60s ./server/ocr/...
func FuzzDecodeResult(f *testing.F) {
	// Seed corpus: a valid document, plus edge cases.
	good, _ := json.Marshal(validResult())
	f.Add(good)
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"document_id":"x"}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`{"document_id":"x","extracted_at":"2026-01-01T00:00:00Z","overall_confidence":2.5,"needs_review":false,"engine":{"name":"tesseract","version":"5"},"fields":{}}`))
	// Polyglot / abuse cases:
	f.Add([]byte("\x00\x00\x00\x00"))
	f.Add([]byte("PK\x03\x04")) // zip header
	f.Add([]byte("\xff\xd8\xff\xe0")) // JPEG SOI
	f.Add([]byte(`{"document_id":"x","overall_confidence":1e308,"engine":{"name":"tesseract","version":"5"},"fields":{},"extracted_at":"2026-01-01T00:00:00Z","needs_review":false}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		r, err := DecodeResult(data)
		if err != nil {
			// Errors are fine. The invariant is "no panic, no half-built
			// invalid Result escapes."
			if r != nil {
				t.Errorf("error path returned non-nil result: %v", r)
			}
			return
		}
		// If decode succeeded, re-validate. Validation must agree with
		// the decode contract.
		if err := r.Validate(); err != nil {
			t.Errorf("DecodeResult accepted but Validate rejected: %v\ninput: %q", err, data)
		}
	})
}
