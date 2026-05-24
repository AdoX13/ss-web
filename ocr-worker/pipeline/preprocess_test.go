package pipeline

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func makeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Fill with a non-uniform pattern so jpeg encoder doesn't optimize away.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatalf("encode test jpeg: %v", err)
	}
	return buf.Bytes()
}

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
	return buf.Bytes()
}

func TestPreprocess_RejectsEmpty(t *testing.T) {
	if _, err := Preprocess(nil); !errors.Is(err, ErrImageDecode) {
		t.Fatalf("nil input: want ErrImageDecode, got %v", err)
	}
	if _, err := Preprocess([]byte{}); !errors.Is(err, ErrImageDecode) {
		t.Fatalf("empty input: want ErrImageDecode, got %v", err)
	}
}

func TestPreprocess_RejectsTextPayload(t *testing.T) {
	_, err := Preprocess([]byte("definitely not an image"))
	if !errors.Is(err, ErrUnsupportedMIME) {
		t.Fatalf("text input: want ErrUnsupportedMIME, got %v", err)
	}
}

func TestPreprocess_RejectsExecutablePolyglot(t *testing.T) {
	// ELF magic — must not pass the MIME allowlist.
	_, err := Preprocess([]byte("\x7fELF\x02\x01\x01\x00"))
	if !errors.Is(err, ErrUnsupportedMIME) {
		t.Fatalf("ELF polyglot: want ErrUnsupportedMIME, got %v", err)
	}
}

func TestPreprocess_AcceptsSmallJPEG(t *testing.T) {
	pp, err := Preprocess(makeJPEG(t, 100, 100))
	if err != nil {
		t.Fatalf("Preprocess small jpeg: %v", err)
	}
	if pp.MIME != "image/jpeg" {
		t.Errorf("mime: want image/jpeg got %s", pp.MIME)
	}
	if pp.Width != 100 || pp.Height != 100 {
		t.Errorf("dims: want 100x100 got %dx%d", pp.Width, pp.Height)
	}
}

func TestPreprocess_AcceptsSmallPNG(t *testing.T) {
	pp, err := Preprocess(makePNG(t, 50, 75))
	if err != nil {
		t.Fatalf("Preprocess small png: %v", err)
	}
	if pp.MIME != "image/png" {
		t.Errorf("mime: want image/png got %s", pp.MIME)
	}
}

func TestPreprocess_DownscalesLarge(t *testing.T) {
	// 6000 wide > TargetMaxPixelsPerSide (4000) but < MaxPixelsPerSide (8000).
	// Should be accepted and downscaled.
	pp, err := Preprocess(makeJPEG(t, 6000, 3000))
	if err != nil {
		t.Fatalf("Preprocess large jpeg: %v", err)
	}
	if pp.OriginalWidth != 6000 || pp.OriginalHeight != 3000 {
		t.Errorf("orig dims: want 6000x3000 got %dx%d", pp.OriginalWidth, pp.OriginalHeight)
	}
	if pp.Width != TargetMaxPixelsPerSide {
		t.Errorf("downscaled width: want %d got %d", TargetMaxPixelsPerSide, pp.Width)
	}
	if pp.Height != 2000 { // 6000:3000 = 4000:2000
		t.Errorf("downscaled height: want 2000 got %d", pp.Height)
	}
}

// Note: we don't test MaxPixelsPerSide rejection with a real >8000px
// image — that would allocate ~250 MB on encode. The Decode path uses
// image.DecodeConfig (header only) before full decode, so this is
// covered by the malformed-header fuzz seeds.
