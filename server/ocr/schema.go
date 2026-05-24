// Package ocr contains the API-side client and result types for the
// sandboxed OCR worker. The OcrResult schema mirrors team plan v3 §6.2.
//
// This package is stdlib-only on purpose: adding a dependency to the API
// server's go.mod for an internal contract would be over-engineering. The
// worker (ocr-worker/) is a separate Go module with its own dependency
// graph (gosseract, etc.).
package ocr

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// EngineName is the only OCR engine we support. The schema's `engine.name`
// field is a JSON `const`; mismatch is a hard validation error.
const EngineName = "tesseract"

// OverallConfidenceReviewThreshold is the floor below which a Result is
// flagged for human review. Mirrors team plan §10 P2 Phase E.
const OverallConfidenceReviewThreshold = 0.95

// ControlType enum values, aligned with the existing Statistics page
// (see server/utils/medical_parser.go for the legacy form).
var ControlTypeValues = []string{
	"Angajare", "Periodic", "Adaptare", "Reluare", "Supraveghere", "Alte",
}

// MedicalOpinion enum values.
var MedicalOpinionValues = []string{
	"APT", "APT Condiționat", "Inapt Temporar", "Inapt",
}

// Result is the worker's response payload.
// It matches the JSON Schema in team plan v3 §6.2.
type Result struct {
	DocumentID        string    `json:"document_id"`
	ExtractedAt       time.Time `json:"extracted_at"`
	OverallConfidence float64   `json:"overall_confidence"`
	NeedsReview       bool      `json:"needs_review"`
	Engine            Engine    `json:"engine"`
	Fields            Fields    `json:"fields"`
	RawText           string    `json:"raw_text,omitempty"`
	ProcessingMs      int       `json:"processing_ms,omitempty"`
}

// Engine identifies the OCR engine that produced the Result.
type Engine struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// BBox is a four-number bounding box [x, y, w, h] in image pixels.
type BBox [4]float64

// Field is a single extracted value with a per-field confidence score.
// Value is a pointer because a missing field is null in the schema, not
// the empty string (those mean different things — the latter means
// Tesseract returned something but the heuristic threw it out).
type Field struct {
	Value      *string `json:"value"`
	Confidence float64 `json:"confidence"`
	BBox       *BBox   `json:"bbox,omitempty"`
}

// EnumField is a Field whose Value is constrained to a fixed set.
// Validation enforces the set; the wire shape is identical to Field.
type EnumField struct {
	Value      *string `json:"value"`
	Confidence float64 `json:"confidence"`
	BBox       *BBox   `json:"bbox,omitempty"`
}

// Fields holds every extracted field. All fields are optional on the
// wire so the worker can omit ones it had no signal for; the API decides
// what's required for persistence.
type Fields struct {
	PatientName    *Field     `json:"patient_name,omitempty"`
	PatientCNP     *Field     `json:"patient_cnp,omitempty"`
	Profession     *Field     `json:"profession,omitempty"`
	Workplace      *Field     `json:"workplace,omitempty"`
	ControlType    *EnumField `json:"control_type,omitempty"`
	MedicalOpinion *EnumField `json:"medical_opinion,omitempty"`
	ExamDate       *Field     `json:"exam_date,omitempty"`
	ExpirationDate *Field     `json:"expiration_date,omitempty"`
	DoctorName     *Field     `json:"doctor_name,omitempty"`
}

// ErrorResponse is the JSON body the worker returns on non-2xx status.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Validate enforces the constraints from the JSON Schema in team plan
// §6.2. It is called by the API on every response and by tests.
//
// Validation is intentionally strict: a worker bug that returns garbage
// must surface as a 422 to the caller, not as a silently-corrupt DB row.
func (r *Result) Validate() error {
	if r.DocumentID == "" {
		return errors.New("document_id is required")
	}
	if r.ExtractedAt.IsZero() {
		return errors.New("extracted_at is required")
	}
	if r.OverallConfidence < 0 || r.OverallConfidence > 1 {
		return fmt.Errorf("overall_confidence %v out of [0, 1]", r.OverallConfidence)
	}
	if r.Engine.Name != EngineName {
		return fmt.Errorf("engine.name must be %q, got %q", EngineName, r.Engine.Name)
	}
	if r.Engine.Version == "" {
		return errors.New("engine.version is required")
	}
	if err := r.Fields.Validate(); err != nil {
		return fmt.Errorf("fields: %w", err)
	}
	// Cross-check the review flag against the threshold.
	expected := r.OverallConfidence < OverallConfidenceReviewThreshold
	if r.NeedsReview != expected {
		return fmt.Errorf(
			"needs_review=%v inconsistent with overall_confidence=%v (threshold %v)",
			r.NeedsReview, r.OverallConfidence, OverallConfidenceReviewThreshold,
		)
	}
	return nil
}

// Validate checks every present field's confidence and enum membership.
func (f *Fields) Validate() error {
	if err := f.PatientName.validateField("patient_name"); err != nil {
		return err
	}
	if err := f.PatientCNP.validateField("patient_cnp"); err != nil {
		return err
	}
	if err := f.Profession.validateField("profession"); err != nil {
		return err
	}
	if err := f.Workplace.validateField("workplace"); err != nil {
		return err
	}
	if err := f.ControlType.validateEnum("control_type", ControlTypeValues); err != nil {
		return err
	}
	if err := f.MedicalOpinion.validateEnum("medical_opinion", MedicalOpinionValues); err != nil {
		return err
	}
	if err := f.ExamDate.validateField("exam_date"); err != nil {
		return err
	}
	if err := f.ExpirationDate.validateField("expiration_date"); err != nil {
		return err
	}
	if err := f.DoctorName.validateField("doctor_name"); err != nil {
		return err
	}
	return nil
}

func (f *Field) validateField(name string) error {
	if f == nil {
		return nil
	}
	if f.Confidence < 0 || f.Confidence > 1 {
		return fmt.Errorf("%s.confidence %v out of [0, 1]", name, f.Confidence)
	}
	return nil
}

func (f *EnumField) validateEnum(name string, allowed []string) error {
	if f == nil {
		return nil
	}
	if f.Confidence < 0 || f.Confidence > 1 {
		return fmt.Errorf("%s.confidence %v out of [0, 1]", name, f.Confidence)
	}
	if f.Value == nil {
		return nil
	}
	for _, v := range allowed {
		if *f.Value == v {
			return nil
		}
	}
	return fmt.Errorf("%s.value %q not in allowed enum", name, *f.Value)
}

// DecodeResult parses and validates a worker response body in one step.
func DecodeResult(body []byte) (*Result, error) {
	var r Result
	dec := json.NewDecoder(bytesReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode ocr result: %w", err)
	}
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ocr result: %w", err)
	}
	return &r, nil
}
