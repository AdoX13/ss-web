package ocr

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func validResult() *Result {
	val := func(s string) *string { return &s }
	conf := 0.97
	return &Result{
		DocumentID:        "doc-1",
		ExtractedAt:       time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC),
		OverallConfidence: conf,
		NeedsReview:       false, // 0.97 >= 0.95 threshold
		Engine:            Engine{Name: EngineName, Version: "5.3.0"},
		Fields: Fields{
			PatientName:    &Field{Value: val("Popescu Ion"), Confidence: 0.99},
			PatientCNP:     &Field{Value: val("1800101123456"), Confidence: 0.98},
			ControlType:    &EnumField{Value: val("Angajare"), Confidence: 0.97},
			MedicalOpinion: &EnumField{Value: val("APT"), Confidence: 0.99},
		},
	}
}

func TestResultValidate_Happy(t *testing.T) {
	if err := validResult().Validate(); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestResultValidate_RejectsBadConfidence(t *testing.T) {
	r := validResult()
	r.OverallConfidence = 1.5
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for overall_confidence > 1")
	}
	r.OverallConfidence = -0.1
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for overall_confidence < 0")
	}
}

func TestResultValidate_RejectsWrongEngine(t *testing.T) {
	r := validResult()
	r.Engine.Name = "easyocr"
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for non-tesseract engine")
	}
}

func TestResultValidate_RejectsBadEnumValue(t *testing.T) {
	r := validResult()
	bad := "TotallyNotARealValue"
	r.Fields.ControlType.Value = &bad
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for invalid control_type enum value")
	}
}

func TestResultValidate_ReviewFlagMustMatchThreshold(t *testing.T) {
	r := validResult()
	r.OverallConfidence = 0.80 // below threshold
	r.NeedsReview = false      // mismatch
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for needs_review/threshold mismatch")
	}
}

func TestDecodeResult_RoundTrip(t *testing.T) {
	r := validResult()
	body, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	back, err := DecodeResult(body)
	if err != nil {
		t.Fatalf("DecodeResult: %v", err)
	}
	if back.DocumentID != r.DocumentID {
		t.Errorf("doc id: want %q got %q", r.DocumentID, back.DocumentID)
	}
	if back.OverallConfidence != r.OverallConfidence {
		t.Errorf("confidence drift: want %v got %v", r.OverallConfidence, back.OverallConfidence)
	}
}

func TestDecodeResult_RejectsUnknownFields(t *testing.T) {
	body := []byte(`{
		"document_id": "x", "extracted_at": "2026-05-24T00:00:00Z",
		"overall_confidence": 0.99, "needs_review": false,
		"engine": {"name": "tesseract", "version": "5"},
		"fields": {},
		"surprise_field": "danger"
	}`)
	if _, err := DecodeResult(body); err == nil ||
		!strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field error, got: %v", err)
	}
}

func TestDecodeResult_RejectsMalformed(t *testing.T) {
	for _, body := range [][]byte{
		nil,
		[]byte(""),
		[]byte("{"),
		[]byte(`{"document_id": 123}`), // wrong type
	} {
		if _, err := DecodeResult(body); err == nil {
			t.Errorf("expected error for %q", string(body))
		}
	}
}
