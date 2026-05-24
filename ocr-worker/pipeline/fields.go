package pipeline

import (
	"regexp"
	"strings"
)

// Field extractor for `medicina muncii` (Romanian occupational medicine
// aptitude certificate) forms.
//
// Strategy
// --------
// The extractor runs label-anchored regexes against the full OCR text
// to find candidate values. For each candidate, it derives a confidence
// score from the Tesseract per-word confidences of the words that
// overlap the matched substring. Words missing from OCROutput (e.g. in
// fake-engine tests) fall back to a configurable default.
//
// The set of fields and the enum values come from team plan v3 §6.2.
// The regexes are derived from the legacy server/utils/medical_parser.go
// but the output shape is the new schema (per-field confidence + bbox).

// ExtractFields runs the full extractor and returns a populated Fields
// struct. Fields with no candidate match are left as nil — the wire
// schema treats nil as "field absent."
//
// fallbackConfidence is used when the engine returned no word-level
// confidences (e.g. fake engines in tests). In production, gosseract
// always provides them, so this rarely fires.
func ExtractFields(ocr *OCROutput, fallbackConfidence float64) Fields {
	text := ocr.Text
	if text == "" {
		return Fields{}
	}
	wc := newWordConfidence(ocr.Words, fallbackConfidence)

	var f Fields

	// Patient identity. Character classes intentionally exclude '\n' —
	// Go regex's `\s` includes newline, which would cause NUME to
	// greedily eat into the following PRENUME label.
	f.PatientName = combineNamePrenume(
		extractField(text, `(?i)NUME[:;]?[ \t]*([A-Za-zĂÂÎȘȚăâîșț \t\-]{2,40})`, wc),
		extractField(text, `(?i)PRENUME[:;]?[ \t]*([A-Za-zĂÂÎȘȚăâîșț \t\-]{2,40})`, wc),
	)

	// CNP — apply checksum; if invalid, treat as missing (no match
	// is better than wrong CNP).
	if cnp := extractField(text, `(?i)CNP[:;]?\s*(\d{13})`, wc); cnp != nil {
		if cnp.Value != nil && ValidateCNP(*cnp.Value) {
			f.PatientCNP = cnp
		}
	}

	// Profession + workplace.
	f.Profession = extractField(text,
		`(?i)Profesie\s*[\/\|]\s*func[tțţ]ie[:;]?\s*([^\n]{2,80})`, wc)
	f.Workplace = extractField(text,
		`(?i)Locul?\s+de\s+munca[:;]?\s*([^\n]{2,80})`, wc)

	// Doctor name — line under the signature. Heuristic: look for
	// "Dr." or "Medic" followed by capitalized words.
	f.DoctorName = extractField(text,
		`(?i)(?:Dr\.?|Medic(?:ul)?)[ \t]+([A-ZĂÂÎȘȚ][A-Za-zĂÂÎȘȚăâîșț \t\.\-]{2,60})`, wc)

	// Control type — find which of the six checkbox labels is the
	// active one. We use the same "checked vs unchecked" gap analysis
	// as the legacy parser, but mapped to the schema enum values.
	f.ControlType = detectControlType(text, wc)

	// Medical opinion — APT / APT Condiționat / Inapt Temporar / Inapt.
	f.MedicalOpinion = detectMedicalOpinion(text, wc)

	// Dates. The exam date is labeled "Data"; the next-exam date is
	// labeled "Data urmatoarei examinari" or "Data urm. examinari".
	f.ExamDate = extractDateField(text,
		`(?i)Data[:;]?\s*([^\n]{6,40})`, wc, []string{"urmatoarei", "urm."})
	f.ExpirationDate = extractDateField(text,
		`(?i)Data\s+urm(?:atoarei)?\.?\s*examinari[:;]?\s*([^\n]{6,40})`, wc, nil)

	return f
}

// extractField runs a single regex and constructs a Field if the
// pattern matched. The captured group becomes Value; confidence is
// derived from the OCR word confidences over the match span.
func extractField(text, pattern string, wc *wordConfidence) *Field {
	re := regexp.MustCompile(pattern)
	loc := re.FindStringSubmatchIndex(text)
	if loc == nil || len(loc) < 4 {
		return nil
	}
	start, end := loc[2], loc[3]
	if start < 0 || end <= start || end > len(text) {
		return nil
	}
	raw := strings.TrimSpace(text[start:end])
	if raw == "" {
		return nil
	}
	conf, bbox := wc.scoreSubstring(raw)
	return &Field{
		Value:      strPtr(raw),
		Confidence: conf,
		BBox:       bbox,
	}
}

// extractDateField is like extractField but also parses the captured
// text as a Romanian date. excludeIfContains lets the caller skip
// matches that look like another labeled date (e.g. the "Data" regex
// would otherwise greedily match "Data urmatoarei examinari").
func extractDateField(text, pattern string, wc *wordConfidence, excludeIfContains []string) *Field {
	re := regexp.MustCompile(pattern)
	loc := re.FindStringSubmatchIndex(text)
	if loc == nil || len(loc) < 4 {
		return nil
	}
	raw := strings.TrimSpace(text[loc[2]:loc[3]])
	for _, ex := range excludeIfContains {
		if strings.Contains(strings.ToLower(raw), ex) {
			return nil
		}
	}
	_, canon, ok := ParseRomanianDate(raw)
	if !ok {
		return nil
	}
	conf, bbox := wc.scoreSubstring(raw)
	return &Field{Value: strPtr(canon), Confidence: conf, BBox: bbox}
}

// combineNamePrenume merges separately-extracted NUME and PRENUME fields
// into a single PatientName field. The combined confidence is the min
// of the two — a noisy first name pulls the whole record into review.
func combineNamePrenume(nume, prenume *Field) *Field {
	switch {
	case nume == nil && prenume == nil:
		return nil
	case nume == nil:
		return prenume
	case prenume == nil:
		return nume
	}
	combined := strings.TrimSpace(*nume.Value) + " " + strings.TrimSpace(*prenume.Value)
	conf := nume.Confidence
	if prenume.Confidence < conf {
		conf = prenume.Confidence
	}
	return &Field{Value: strPtr(combined), Confidence: conf}
}

// detectControlType maps the existing 6-way checkbox to the §6.2 enum.
// We look for each label in the OCR text and check whether the gap to
// the next label contains an empty-box pattern; the active label is
// the one without an empty box.
//
// Legacy labels (medical_parser.go) → schema enum:
//
//	Angajare              → "Angajare"
//	Control               → "Periodic"  (label is "Control medical periodic")
//	Adaptare              → "Adaptare"
//	Reluarea              → "Reluare"   (label is "Reluarea muncii")
//	Supraveghere          → "Supraveghere"
//	Alte                  → "Alte"
func detectControlType(text string, wc *wordConfidence) *Field {
	// Fuzzy OCR normalization mirrors the legacy parser's known quirks.
	t := strings.ReplaceAll(text, "Roluarca", "Reluarea")
	t = strings.ReplaceAll(t, "Ane", "Alte")
	rowStart := strings.Index(t, "Angajare")
	if rowStart == -1 {
		return nil
	}
	row := t[rowStart:]
	pairs := []struct {
		label, next, enumValue string
	}{
		{"Angajare", "Control", "Angajare"},
		{"Control", "Adaptare", "Periodic"},
		{"Adaptare", "Reluarea", "Adaptare"},
		{"Reluarea", "Supraveghere", "Reluare"},
		{"Supraveghere", "Alte", "Supraveghere"},
		{"Alte", "", "Alte"},
	}
	for _, p := range pairs {
		if isBoxChecked(row, p.label, p.next) {
			conf, bbox := wc.scoreSubstring(p.label)
			return &Field{Value: strPtr(p.enumValue), Confidence: conf, BBox: bbox}
		}
	}
	return nil
}

// detectMedicalOpinion picks one of APT / APT Condiționat / Inapt
// Temporar / Inapt using the same gap-analysis approach.
func detectMedicalOpinion(text string, wc *wordConfidence) *Field {
	// Normalize known OCR garbles.
	t := strings.ReplaceAll(text, "ApT", "APT")
	avizStart := strings.Index(t, "AVIZ MEDICAL")
	if avizStart == -1 {
		// Fall back to scanning the whole document.
		avizStart = 0
	}
	row := t[avizStart:]

	pairs := []struct {
		label, next, enumValue string
	}{
		{"APT", "CONDITIONAT", "APT"},
		{"CONDITIONAT", "INAPT TEMPORAR", "APT Condiționat"},
		{"INAPT TEMPORAR", "INAPT", "Inapt Temporar"},
		{"INAPT", "", "Inapt"},
	}
	for _, p := range pairs {
		if !strings.Contains(strings.ToUpper(row), p.label) {
			continue
		}
		// Case-fold lookup but check on uppercase form for consistency.
		upper := strings.ToUpper(row)
		if isBoxChecked(upper, p.label, p.next) {
			conf, bbox := wc.scoreSubstring(p.label)
			return &Field{Value: strPtr(p.enumValue), Confidence: conf, BBox: bbox}
		}
	}
	return nil
}

// isBoxChecked returns true if the gap between labelA and labelB
// contains no empty-checkbox pattern. Lifted from
// server/utils/medical_parser.go and simplified.
func isBoxChecked(text, labelA, labelB string) bool {
	curIdx := strings.Index(text, labelA)
	if curIdx == -1 {
		return false
	}
	searchStart := curIdx + len(labelA)
	var gap string
	if labelB == "" {
		end := searchStart + 20
		if end > len(text) {
			end = len(text)
		}
		gap = text[searchStart:end]
	} else {
		nextIdx := strings.Index(text[searchStart:], labelB)
		if nextIdx == -1 {
			end := searchStart + 20
			if end > len(text) {
				end = len(text)
			}
			gap = text[searchStart:end]
		} else {
			gap = text[searchStart : searchStart+nextIdx]
		}
	}
	emptyBoxRe := regexp.MustCompile(`\[\s*[\[\]\-\|]?\s*\]`)
	return !emptyBoxRe.MatchString(gap)
}
