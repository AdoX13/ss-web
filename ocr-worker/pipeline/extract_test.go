package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeEngine satisfies OCREngine for tests. The Recognize function lets
// each test inject the OCR output (or a failure) it needs.
type fakeEngine struct {
	out *OCROutput
	err error
}

func (f *fakeEngine) Recognize(_ []byte) (*OCROutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}
func (f *fakeEngine) Close() error { return nil }

func TestExtract_HappyPath(t *testing.T) {
	cnp := "180010122551" + string('0'+byte(computeControlDigit("180010122551")))
	text := "NUME: POPESCU\nPRENUME: ION\nCNP: " + cnp +
		"\nAngajare [X] Control [] Adaptare [] Reluarea [] Supraveghere [] Alte []\n" +
		"Data: 01.06.2026"

	in := ExtractInput{
		DocumentID:    "doc-xyz",
		ImageBytes:    makePNG(t, 50, 50),
		Engine:        &fakeEngine{out: fakeOCR(text, 0.97)},
		EngineVersion: "tesseract-5.3",
		Now:           func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	}
	r, err := Extract(context.Background(), in)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if r.DocumentID != "doc-xyz" {
		t.Errorf("doc id: got %s", r.DocumentID)
	}
	if r.Engine.Name != EngineName {
		t.Errorf("engine.name: got %s", r.Engine.Name)
	}
	if r.Fields.PatientCNP == nil {
		t.Errorf("expected CNP populated")
	}
}

func TestExtract_BubblesPreprocessError(t *testing.T) {
	_, err := Extract(context.Background(), ExtractInput{
		DocumentID:    "doc",
		ImageBytes:    []byte("not an image"),
		Engine:        &fakeEngine{},
		EngineVersion: "v",
		Now:           time.Now,
	})
	if !errors.Is(err, ErrUnsupportedMIME) {
		t.Fatalf("expected ErrUnsupportedMIME, got %v", err)
	}
}

func TestExtract_BubblesEngineError(t *testing.T) {
	_, err := Extract(context.Background(), ExtractInput{
		DocumentID:    "doc",
		ImageBytes:    makePNG(t, 20, 20),
		Engine:        &fakeEngine{err: errors.New("kaboom")},
		EngineVersion: "v",
		Now:           time.Now,
	})
	if err == nil || err.Error() == "" {
		t.Fatal("expected engine error to bubble")
	}
}

func TestExtract_NoFieldsExtracted(t *testing.T) {
	_, err := Extract(context.Background(), ExtractInput{
		DocumentID:    "doc",
		ImageBytes:    makePNG(t, 20, 20),
		Engine:        &fakeEngine{out: fakeOCR("totally unstructured text with no labels", 0.9)},
		EngineVersion: "v",
		Now:           time.Now,
	})
	if !errors.Is(err, ErrNoFieldsExtracted) {
		t.Fatalf("expected ErrNoFieldsExtracted, got %v", err)
	}
}

func TestExtract_RejectsMissingDocumentID(t *testing.T) {
	_, err := Extract(context.Background(), ExtractInput{
		DocumentID:    "",
		ImageBytes:    makePNG(t, 20, 20),
		Engine:        &fakeEngine{},
		EngineVersion: "v",
	})
	if err == nil {
		t.Fatal("expected error for empty document_id")
	}
}

func TestExtract_HonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Extract(ctx, ExtractInput{
		DocumentID:    "doc",
		ImageBytes:    makePNG(t, 20, 20),
		Engine:        &fakeEngine{},
		EngineVersion: "v",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
