package pipeline

import (
	"bytes"
	"compress/gzip"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

// Go-native fuzz tests for the OCR worker. Required by Lab 10 Ex 1.
//
// Each Fuzz* function below asserts an invariant rather than a value:
//
//   - Preprocess must never panic and must never return a non-nil
//     PreprocessedImage when it also returns an error.
//   - ExtractFields must never panic on arbitrary OCR text and must
//     never return a Field whose Confidence is outside [0, 1].
//   - ValidateCNP must never panic and must agree with the inverse
//     property: a CNP it accepts must satisfy the control digit
//     equation when re-derived.
//
// Run with (per Lab 10 fuzztime budget):
//
//	go test -fuzz=FuzzPreprocess -fuzztime=60s ./pipeline/...
//	go test -fuzz=FuzzExtractFields -fuzztime=60s ./pipeline/...
//	go test -fuzz=FuzzValidateCNP -fuzztime=60s ./pipeline/...

// ----------- Preprocessor fuzz -----------

// FuzzPreprocess feeds arbitrary bytes into the preprocessor. The
// invariant is "no panic, and a well-formed result whenever err is nil."
func FuzzPreprocess(f *testing.F) {
	// Seed corpus mixes valid images, malformed images, polyglots, and
	// known-bad header patterns. Lab 10 §10 P2 Phase F calls these out
	// explicitly.
	f.Add(seedJPEG(f, 30, 30))
	f.Add(seedPNG(f, 30, 30))
	f.Add([]byte{}) // empty
	f.Add([]byte("not an image"))
	// Truncated JPEG SOI (no EOI):
	f.Add([]byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10})
	// Truncated PNG header (signature only):
	f.Add([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	// "Zip bomb" stand-in: gzip with a giant declared size — must not
	// be confused for an image.
	f.Add(makeFakeGzip(f))
	// Polyglot: ZIP header (PK\x03\x04) — not in MIME allowlist.
	f.Add([]byte("PK\x03\x04 zip-as-image"))
	// EXIF-only JPEG header without scan data.
	f.Add([]byte("\xff\xd8\xff\xe1\x00\x10Exif\x00\x00"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Cap inputs to a reasonable size — fuzzing with 100 MB blobs
		// would burn CPU on the LimitReader path, not on the actual
		// decode logic we care about. The boundary cap is tested
		// separately in TestPreprocess_*.
		if len(data) > 2*1024*1024 {
			data = data[:2*1024*1024]
		}
		pp, err := Preprocess(data)
		if err == nil && pp == nil {
			t.Fatalf("nil result with nil error")
		}
		if err != nil && pp != nil {
			t.Fatalf("non-nil result with non-nil error: %v", err)
		}
		if pp != nil {
			if pp.Width <= 0 || pp.Height <= 0 {
				t.Errorf("non-positive dims: %dx%d", pp.Width, pp.Height)
			}
			if len(pp.EncodedBytes) == 0 {
				t.Errorf("empty re-encoded bytes")
			}
		}
	})
}

// ----------- Field extractor fuzz -----------

// FuzzExtractFields throws arbitrary text at the medical-form
// extractor. We don't care what fields it returns; we only care that
// it doesn't panic and never returns out-of-range confidences.
func FuzzExtractFields(f *testing.F) {
	// Seeds mix "real-looking" text, empty input, single labels,
	// pathological repetition, control characters.
	f.Add("NUME: TEST\nPRENUME: ION\nCNP: 1800101225518\nData: 01.01.2026")
	f.Add("")
	f.Add(strings.Repeat("Angajare [] Control [] Adaptare [] Reluarea [] ", 200))
	f.Add("CNP: ")
	f.Add("AVIZ MEDICAL\nAPT [X] CONDITIONAT [] INAPT TEMPORAR [] INAPT []")
	f.Add(string([]byte{0, 1, 2, 3, 4, 5, 0xff, 0xfe}))
	f.Add("NUME:" + strings.Repeat("A", 10000))
	f.Add("Profesie / functie: " + strings.Repeat("X ", 500))

	f.Fuzz(func(t *testing.T, text string) {
		// Cap input length — extremely long strings just exercise
		// regex backtracking, not extractor logic.
		if len(text) > 100*1024 {
			text = text[:100*1024]
		}
		fields := ExtractFields(&OCROutput{Text: text}, 0.5)
		checkField(t, "patient_name", fields.PatientName)
		checkField(t, "patient_cnp", fields.PatientCNP)
		checkField(t, "profession", fields.Profession)
		checkField(t, "workplace", fields.Workplace)
		checkField(t, "control_type", fields.ControlType)
		checkField(t, "medical_opinion", fields.MedicalOpinion)
		checkField(t, "exam_date", fields.ExamDate)
		checkField(t, "expiration_date", fields.ExpirationDate)
		checkField(t, "doctor_name", fields.DoctorName)

		// If CNP came back, its value must satisfy the checksum —
		// the extractor is supposed to drop invalid ones.
		if fields.PatientCNP != nil {
			if v := fields.PatientCNP.Value; v == nil || !ValidateCNP(*v) {
				t.Errorf("extractor returned non-checksum-valid CNP: %v", v)
			}
		}
	})
}

func checkField(t *testing.T, name string, f *Field) {
	t.Helper()
	if f == nil {
		return
	}
	if f.Confidence < 0 || f.Confidence > 1 {
		t.Errorf("%s: confidence %v out of [0, 1]", name, f.Confidence)
	}
	if f.Value == nil {
		// A non-nil Field with nil Value is allowed by the wire schema
		// only if the worker explicitly wants to say "I found a label
		// but no value." Our extractor never returns that — flag it.
		t.Errorf("%s: non-nil field with nil value", name)
	}
}

// ----------- CNP validator fuzz -----------

// FuzzValidateCNP feeds arbitrary strings into the CNP validator. We
// assert: no panic, and any accepted string is genuinely 13 digits
// satisfying the checksum.
func FuzzValidateCNP(f *testing.F) {
	f.Add("1800101225518")
	f.Add("0000000000000")
	f.Add("")
	f.Add("12345")
	f.Add("18001012255180") // 14 chars
	f.Add("1800101abc5518")
	f.Add("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")

	f.Fuzz(func(t *testing.T, s string) {
		ok := ValidateCNP(s)
		if !ok {
			return
		}
		if len(s) != 13 {
			t.Errorf("accepted %q with len %d", s, len(s))
		}
		for i := 0; i < len(s); i++ {
			if s[i] < '0' || s[i] > '9' {
				t.Errorf("accepted non-digit at %d: %q", i, s)
				return
			}
		}
		// Re-derive control digit and compare.
		sum := 0
		for i := 0; i < 12; i++ {
			sum += int(s[i]-'0') * CNPControlConstants[i]
		}
		c := sum % 11
		if c == 10 {
			c = 1
		}
		if c != int(s[12]-'0') {
			t.Errorf("accepted CNP with wrong control digit: %q", s)
		}
	})
}

// ----------- Date parser fuzz -----------

// FuzzParseRomanianDate asserts no-panic and that any accepted date
// round-trips through ParseRomanianDate(canonical) == accepted.
func FuzzParseRomanianDate(f *testing.F) {
	f.Add("01.06.2026")
	f.Add("1/6/2026")
	f.Add("15 ianuarie 2026")
	f.Add("31.02.2026") // invalid
	f.Add("")
	f.Add("00.00.0000")
	f.Add("99.99.9999")
	f.Add(strings.Repeat("9", 1000))

	f.Fuzz(func(t *testing.T, s string) {
		ti, canon, ok := ParseRomanianDate(s)
		if !ok {
			return
		}
		// Accepted input — canonical form must round-trip.
		if canon == "" {
			t.Errorf("accepted but empty canonical: %q", s)
			return
		}
		ti2, canon2, ok2 := ParseRomanianDate(canon)
		if !ok2 || !ti.Equal(ti2) || canon != canon2 {
			t.Errorf("canonical %q does not round-trip from input %q", canon, s)
		}
	})
}

// ----------- helpers -----------

func seedJPEG(_ *testing.F, w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 128, 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func seedPNG(_ *testing.F, w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func makeFakeGzip(_ *testing.F) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, _ = gw.Write(bytes.Repeat([]byte("A"), 4096))
	_ = gw.Close()
	return buf.Bytes()
}
