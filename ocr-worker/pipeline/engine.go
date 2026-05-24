package pipeline

// OCREngine is the abstraction over Tesseract that the rest of the
// pipeline depends on. Keeping the rest of the package free of any
// direct Tesseract import means we can:
//
//   1. Run tests against a fake engine (no libtesseract install needed).
//   2. Swap engines later (e.g. PaddleOCR) without touching the field
//      extractor or the preprocess layer.
//
// Implementations live in engine_tesseract.go (CGO build) and
// engine_stub.go (CGO-disabled build).
type OCREngine interface {
	// Recognize runs OCR on a single image. Input is the encoded image
	// bytes (PNG, post-Preprocess); output is the full text plus a
	// per-word list with bounding boxes and confidence in [0, 1].
	//
	// Implementations must be safe to call from multiple goroutines.
	// Tesseract's underlying client is not goroutine-safe, so the pool
	// implementation serializes access internally.
	Recognize(image []byte) (*OCROutput, error)

	// Close releases any underlying handles (Tesseract clients).
	Close() error
}

// OCROutput is the engine-agnostic OCR result.
type OCROutput struct {
	Text  string
	Words []Word
}

// Word is a single token with its bounding box and confidence.
//
// Confidence is normalized to [0, 1]. Tesseract natively returns
// confidence in 0..100; the engine adapter does the conversion so
// downstream code never has to think about scale.
type Word struct {
	Text       string
	Confidence float64
	BBox       BBox
}
