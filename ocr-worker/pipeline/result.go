// Package pipeline runs the worker-side OCR pipeline: preprocess →
// Tesseract → field extraction. The Result types declared here mirror
// the wire schema in team plan v3 §6.2 and the API-side types in
// server/ocr/schema.go.
//
// The two modules are intentionally separate (worker has CGO deps; the
// API server should not). They are kept in sync by a shared JSON
// fixture and a wire round-trip test in tests/integration/.
package pipeline

const (
	EngineName = "tesseract"

	// ReviewThreshold is the minimum overall_confidence at which a Result
	// is treated as authoritative. Below this, the API must route to the
	// review queue (see confidence.Compute).
	ReviewThreshold = 0.95
)

// Result is the worker response body. JSON tags match team plan §6.2.
type Result struct {
	DocumentID        string  `json:"document_id"`
	ExtractedAt       string  `json:"extracted_at"`
	OverallConfidence float64 `json:"overall_confidence"`
	NeedsReview       bool    `json:"needs_review"`
	Engine            Engine  `json:"engine"`
	Fields            Fields  `json:"fields"`
	RawText           string  `json:"raw_text,omitempty"`
	ProcessingMs      int     `json:"processing_ms,omitempty"`
}

// Engine identifies which OCR engine ran.
type Engine struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// BBox is [x, y, w, h] in source-image pixels.
type BBox [4]float64

// Field is a single extracted value with confidence.
type Field struct {
	Value      *string `json:"value"`
	Confidence float64 `json:"confidence"`
	BBox       *BBox   `json:"bbox,omitempty"`
}

// Fields holds every extracted field. All optional; the worker omits
// fields it couldn't locate.
type Fields struct {
	PatientName    *Field `json:"patient_name,omitempty"`
	PatientCNP     *Field `json:"patient_cnp,omitempty"`
	Profession     *Field `json:"profession,omitempty"`
	Workplace      *Field `json:"workplace,omitempty"`
	ControlType    *Field `json:"control_type,omitempty"`
	MedicalOpinion *Field `json:"medical_opinion,omitempty"`
	ExamDate       *Field `json:"exam_date,omitempty"`
	ExpirationDate *Field `json:"expiration_date,omitempty"`
	DoctorName     *Field `json:"doctor_name,omitempty"`
}

// strPtr returns a pointer to the given string. Used to keep field
// construction terse — Field.Value is *string because nil and "" mean
// different things in the wire schema.
func strPtr(s string) *string { return &s }
