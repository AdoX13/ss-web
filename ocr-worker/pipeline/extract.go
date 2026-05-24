package pipeline

import (
	"context"
	"fmt"
	"time"
)

// ExtractInput bundles the orchestrator's inputs. A struct (instead of
// positional args) keeps main.go readable and lets us add optional
// fields without breaking the signature.
type ExtractInput struct {
	DocumentID    string
	ImageBytes    []byte
	Engine        OCREngine
	EngineVersion string
	// Now is the clock; tests inject a fixed value. main.go uses time.Now.
	Now func() time.Time
}

// Extract is the worker-side orchestrator: preprocess → OCR → field
// extraction. Returns a Result with Fields populated but
// OverallConfidence and NeedsReview unset — the caller is expected to
// compute those last so the values reflect the final Fields map.
//
// Errors are wrapped sentinels (ErrUnsupportedMIME, ErrImageDecode,
// ErrImageTooLarge, ErrTesseract, ErrNoFieldsExtracted) so main.go can
// route them to HTTP statuses.
func Extract(ctx context.Context, in ExtractInput) (*Result, error) {
	if in.DocumentID == "" {
		return nil, fmt.Errorf("document_id is required")
	}
	if in.Engine == nil {
		return nil, fmt.Errorf("ocr engine is required")
	}
	if in.Now == nil {
		in.Now = time.Now
	}

	// Honor cancellation at the boundary between stages — Tesseract
	// itself doesn't accept a context, but we can avoid starting work
	// after a deadline expired.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	pp, err := Preprocess(in.ImageBytes)
	if err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ocr, err := in.Engine.Recognize(pp.EncodedBytes)
	if err != nil {
		return nil, err
	}

	fields := ExtractFields(ocr, 0.5)
	if !hasAnyField(fields) {
		return nil, fmt.Errorf("%w: ocr produced no recognizable fields", ErrNoFieldsExtracted)
	}

	return &Result{
		DocumentID:  in.DocumentID,
		ExtractedAt: in.Now().UTC().Format(time.RFC3339Nano),
		Engine: Engine{
			Name:    EngineName,
			Version: in.EngineVersion,
		},
		Fields:  fields,
		RawText: ocr.Text,
	}, nil
}

func hasAnyField(f Fields) bool {
	return f.PatientName != nil ||
		f.PatientCNP != nil ||
		f.Profession != nil ||
		f.Workplace != nil ||
		f.ControlType != nil ||
		f.MedicalOpinion != nil ||
		f.ExamDate != nil ||
		f.ExpirationDate != nil ||
		f.DoctorName != nil
}
