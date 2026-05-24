package confidence

import (
	"testing"

	"medsec-ocr/ocr-worker/pipeline"
)

func field(v string, c float64) *pipeline.Field {
	return &pipeline.Field{Value: &v, Confidence: c}
}

func fullFields() pipeline.Fields {
	return pipeline.Fields{
		PatientName:    field("X", 0.99),
		PatientCNP:     field("Y", 0.98),
		ExamDate:       field("01.01.2026", 0.97),
		ExpirationDate: field("01.01.2027", 0.96),
		ControlType:    field("Angajare", 0.95),
		MedicalOpinion: field("APT", 0.99),
	}
}

func TestCompute_AllAboveThreshold(t *testing.T) {
	overall, review := Compute(fullFields())
	if overall != 0.95 {
		t.Errorf("overall: want 0.95 got %v", overall)
	}
	if review {
		t.Errorf("needs_review: want false")
	}
}

func TestCompute_OneBelowThreshold(t *testing.T) {
	f := fullFields()
	f.PatientName.Confidence = 0.80
	overall, review := Compute(f)
	if overall != 0.80 {
		t.Errorf("overall: want 0.80 got %v", overall)
	}
	if !review {
		t.Errorf("needs_review: want true")
	}
}

func TestCompute_MissingRequiredFieldForcesReview(t *testing.T) {
	f := fullFields()
	f.PatientCNP = nil
	overall, review := Compute(f)
	if overall != 0 {
		t.Errorf("overall: want 0 when required missing, got %v", overall)
	}
	if !review {
		t.Errorf("needs_review: want true when required missing")
	}
}

func TestCompute_AtThresholdIsNotReview(t *testing.T) {
	f := fullFields()
	for _, fld := range []*pipeline.Field{
		f.PatientName, f.PatientCNP, f.ExamDate, f.ExpirationDate,
		f.ControlType, f.MedicalOpinion,
	} {
		fld.Confidence = ReviewThreshold
	}
	overall, review := Compute(f)
	if overall != ReviewThreshold {
		t.Errorf("overall: got %v", overall)
	}
	if review {
		t.Errorf("0.95 is the cutoff — exactly equal is not under threshold")
	}
}
