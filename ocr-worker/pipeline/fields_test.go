package pipeline

import "testing"

// fakeOCR generates an OCROutput with one Word per whitespace-separated
// token in the text, all at the given confidence. Used to exercise the
// field extractor without Tesseract.
func fakeOCR(text string, conf float64) *OCROutput {
	out := &OCROutput{Text: text}
	for _, tk := range tokenize(text) {
		out.Words = append(out.Words, Word{
			Text:       tk,
			Confidence: conf,
			BBox:       BBox{0, 0, 1, 1},
		})
	}
	return out
}

func TestExtractFields_HappyPath(t *testing.T) {
	// Synthetic medicina-muncii text. The CNP control digit is
	// pre-computed (1800101 birthdate, county 22, sequence 551).
	cnp := "180010122551" + string('0'+byte(computeControlDigit("180010122551")))
	text := `UNITATEA MEDICALA: Centrul Medical Test
ADRESA: Str Test 1
TEL: 0211234567

FISA DE APTITUDINE NR. 42

Angajare [X] Control [] Adaptare [] Reluarea [] Supraveghere [] Alte []

NUME: POPESCU
PRENUME: ION
CNP: ` + cnp + `

Profesie / functie: Inginer Software
Locul de munca: Birou IT

AVIZ MEDICAL
APT [X] CONDITIONAT [] INAPT TEMPORAR [] INAPT []

Dr. Ionescu Maria
Data: 15.06.2026
Data urmatoarei examinari: 15.06.2027`

	fields := ExtractFields(fakeOCR(text, 0.97), 0.5)

	if fields.PatientName == nil || *fields.PatientName.Value == "" {
		t.Errorf("PatientName missing")
	}
	if fields.PatientCNP == nil || *fields.PatientCNP.Value != cnp {
		t.Errorf("CNP: want %q got %v", cnp, fields.PatientCNP)
	}
	if fields.Profession == nil || *fields.Profession.Value != "Inginer Software" {
		t.Errorf("Profession: got %v", fields.Profession)
	}
	if fields.Workplace == nil || *fields.Workplace.Value != "Birou IT" {
		t.Errorf("Workplace: got %v", fields.Workplace)
	}
	if fields.ControlType == nil || *fields.ControlType.Value != "Angajare" {
		t.Errorf("ControlType: want Angajare got %v", fields.ControlType)
	}
	if fields.MedicalOpinion == nil || *fields.MedicalOpinion.Value != "APT" {
		t.Errorf("MedicalOpinion: want APT got %v", fields.MedicalOpinion)
	}
	if fields.ExamDate == nil || *fields.ExamDate.Value != "15.06.2026" {
		t.Errorf("ExamDate: want 15.06.2026 got %v", fields.ExamDate)
	}
	if fields.ExpirationDate == nil || *fields.ExpirationDate.Value != "15.06.2027" {
		t.Errorf("ExpirationDate: want 15.06.2027 got %v", fields.ExpirationDate)
	}
}

func TestExtractFields_RejectsBadCNP(t *testing.T) {
	text := "NUME: Test\nPRENUME: Ion\nCNP: 1234567890123\n"
	fields := ExtractFields(fakeOCR(text, 0.99), 0.5)
	if fields.PatientCNP != nil {
		t.Errorf("expected CNP to be rejected by checksum, got %v", *fields.PatientCNP.Value)
	}
}

func TestExtractFields_EmptyText(t *testing.T) {
	fields := ExtractFields(&OCROutput{Text: ""}, 0.5)
	// All fields should be nil.
	if fields.PatientName != nil || fields.PatientCNP != nil || fields.ExamDate != nil {
		t.Errorf("expected empty Fields, got %+v", fields)
	}
}

func TestWordConfidence_MinAggregation(t *testing.T) {
	// Two words contribute to the substring; the min confidence wins.
	wc := newWordConfidence([]Word{
		{Text: "ALPHA", Confidence: 0.95, BBox: BBox{0, 0, 10, 5}},
		{Text: "BETA", Confidence: 0.60, BBox: BBox{10, 0, 8, 5}},
		{Text: "GAMMA", Confidence: 0.99, BBox: BBox{0, 5, 10, 5}},
	}, 0.5)

	conf, bbox := wc.scoreSubstring("ALPHA BETA")
	if conf != 0.60 {
		t.Errorf("conf: want 0.60 got %v", conf)
	}
	if bbox == nil || bbox[0] != 0 || bbox[1] != 0 || bbox[2] != 18 || bbox[3] != 5 {
		t.Errorf("bbox: got %v", bbox)
	}
}

func TestWordConfidence_FallbackWhenNoWords(t *testing.T) {
	wc := newWordConfidence(nil, 0.42)
	conf, bbox := wc.scoreSubstring("anything")
	if conf != 0.42 || bbox != nil {
		t.Errorf("expected fallback (0.42, nil), got (%v, %v)", conf, bbox)
	}
}

func TestTokenize(t *testing.T) {
	got := tokenize("Hello,  World! 123  ăț")
	want := []string{"Hello", "World", "123", "ăț"}
	if len(got) != len(want) {
		t.Fatalf("len: want %d got %d (%v)", len(want), len(got), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d]: want %q got %q", i, want[i], got[i])
		}
	}
}
