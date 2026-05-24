package pipeline

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"

	// Side-effect imports for image format detection.
	_ "image/gif"
)

// Hard limits enforced before any heavy work happens.
const (
	// MaxPixelsPerSide is the upper bound on either width or height.
	// 8000 is generous for medical certificates (A4 @ 600 DPI is ~4960
	// pixels on the long edge) but cheap to enforce.
	MaxPixelsPerSide = 8000

	// MaxTotalPixels caps total area, defending against e.g. 10000×100
	// pathological strips that pass MaxPixelsPerSide individually.
	MaxTotalPixels = 8000 * 8000

	// TargetMaxPixelsPerSide is the size we downscale to before OCR.
	// Tesseract gives diminishing returns above 300 DPI on document
	// scans; this puts the long edge at roughly 4000 pixels.
	TargetMaxPixelsPerSide = 4000
)

// allowedMIME is the boundary allowlist. PNG and JPEG cover everything
// the browser camera and the Python ingestion script produce.
var allowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
}

// PreprocessedImage is the output of Preprocess: a validated, decoded,
// possibly downscaled image ready for Tesseract.
type PreprocessedImage struct {
	// EncodedBytes is the image re-encoded as PNG for handoff to
	// Tesseract via SetImageFromBytes. We re-encode rather than pass the
	// original bytes so that anything Tesseract sees has already passed
	// through Go's image decoder — a much narrower attack surface than
	// libleptonica's loaders.
	EncodedBytes []byte

	// Width and Height are the dimensions of EncodedBytes (post-resize).
	Width  int
	Height int

	// OriginalWidth and OriginalHeight are the source dimensions.
	OriginalWidth  int
	OriginalHeight int

	// MIME is the sniffed MIME of the input.
	MIME string
}

// Preprocess validates and normalizes an image. It enforces the boundary
// caps (MIME allowlist, pixel limits) and downscales oversized images.
//
// Failures are wrapped sentinel errors (ErrUnsupportedMIME, ErrImageTooLarge,
// ErrImageDecode) so main.go can map them onto specific HTTP statuses.
func Preprocess(raw []byte) (*PreprocessedImage, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: empty body", ErrImageDecode)
	}

	mime := http.DetectContentType(raw)
	if !allowedMIME[mime] {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedMIME, mime)
	}

	// Cheap header-only decode first: bails on malformed input before
	// allocating a full pixmap.
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrImageDecode, err)
	}
	if cfg.Width > MaxPixelsPerSide || cfg.Height > MaxPixelsPerSide {
		return nil, fmt.Errorf("%w: %dx%d exceeds %d per side",
			ErrImageTooLarge, cfg.Width, cfg.Height, MaxPixelsPerSide)
	}
	if int64(cfg.Width)*int64(cfg.Height) > MaxTotalPixels {
		return nil, fmt.Errorf("%w: %dx%d total area > %d",
			ErrImageTooLarge, cfg.Width, cfg.Height, MaxTotalPixels)
	}

	// Full decode. The header-only check above limits the damage if a
	// decoder bug were to over-allocate.
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrImageDecode, err)
	}

	origW, origH := img.Bounds().Dx(), img.Bounds().Dy()
	resized := img
	w, h := origW, origH
	if maxSide(origW, origH) > TargetMaxPixelsPerSide {
		resized = downscale(img, TargetMaxPixelsPerSide)
		w, h = resized.Bounds().Dx(), resized.Bounds().Dy()
	}

	// Re-encode as PNG. PNG is lossless, so we don't degrade text
	// edges. JPEG re-encoding would.
	var buf bytes.Buffer
	if err := png.Encode(&buf, resized); err != nil {
		return nil, fmt.Errorf("%w: re-encode: %v", ErrImageDecode, err)
	}

	return &PreprocessedImage{
		EncodedBytes:   buf.Bytes(),
		Width:          w,
		Height:         h,
		OriginalWidth:  origW,
		OriginalHeight: origH,
		MIME:           mime,
	}, nil
}

// EncodeProbe is an indirect way for tests to ensure jpeg/png encoders
// are linked. Without referencing them somewhere, the side-effect
// imports above can be misread as unused by readers.
var _ = jpeg.Decode

// maxSide returns the larger of width/height.
func maxSide(w, h int) int {
	if w > h {
		return w
	}
	return h
}

// downscale resizes the image so its longest side equals target. Uses
// a simple nearest-neighbor sampler — Tesseract handles its own
// interpolation downstream, so we don't need a fancy filter here.
//
// Returns a freshly allocated *image.RGBA. The source image is not
// retained.
func downscale(src image.Image, target int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return src
	}
	var newW, newH int
	if w >= h {
		newW = target
		newH = h * target / w
	} else {
		newH = target
		newW = w * target / h
	}
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		srcY := bounds.Min.Y + y*h/newH
		for x := 0; x < newW; x++ {
			srcX := bounds.Min.X + x*w/newW
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}
