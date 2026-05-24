// Package confidence aggregates per-field confidences into the
// overall_confidence + needs_review pair on an OcrResult.
//
// Per team plan v3 §10 P2 Phase E:
//
//	overall_confidence = min(confidence over required fields present)
//	needs_review       = overall_confidence < 0.95
//
// Required fields are the ones that drive medicina-muncii business
// reports: patient name, CNP, exam date, expiration date, control
// type, medical opinion. Missing required fields force the record into
// review regardless of the present-field confidences.
package confidence

import "medsec-ocr/ocr-worker/pipeline"

// ReviewThreshold mirrors pipeline.ReviewThreshold but is re-exported
// here so callers don't have to import the pipeline package for a
// single constant.
const ReviewThreshold = pipeline.ReviewThreshold

// Compute returns (overall_confidence, needs_review) for a populated
// Fields struct. If any required field is missing, overall_confidence
// is 0 and needs_review is true.
//
// Required fields list comes from §6.2 of the team plan combined with
// the report queries we plan to run on `medical_records` (patient,
// CNP, dates, control type, opinion).
func Compute(f pipeline.Fields) (overall float64, needsReview bool) {
	required := []*pipeline.Field{
		f.PatientName,
		f.PatientCNP,
		f.ExamDate,
		f.ExpirationDate,
		f.ControlType,
		f.MedicalOpinion,
	}
	min := 1.0
	allPresent := true
	for _, fld := range required {
		if fld == nil {
			allPresent = false
			continue
		}
		if fld.Confidence < min {
			min = fld.Confidence
		}
	}
	if !allPresent {
		return 0, true
	}
	return min, min < ReviewThreshold
}
