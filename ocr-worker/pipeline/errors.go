package pipeline

import "errors"

// Sentinel errors returned by the pipeline. main.go maps these onto
// HTTP status codes; tests assert via errors.Is.
var (
	ErrUnsupportedMIME   = errors.New("unsupported image MIME type")
	ErrImageTooLarge     = errors.New("image dimensions exceed limit")
	ErrImageDecode       = errors.New("image decode failed")
	ErrNoFieldsExtracted = errors.New("no recognizable fields in image")
	ErrTesseract         = errors.New("tesseract engine failed")
)
