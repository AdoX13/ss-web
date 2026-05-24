//go:build cgo

package pipeline

import (
	"fmt"
	"sync"

	"github.com/otiai10/gosseract/v2"
)

// TesseractPool is a fixed-size pool of gosseract clients.
//
// gosseract.Client wraps a Tesseract API handle that is NOT goroutine-safe.
// We serialize per-client by holding one mutex per client and acquire a
// client from the pool via a buffered channel. Pool size matches the
// HTTP concurrency cap in main.go so a single request can always grab
// one without contention.
type TesseractPool struct {
	clients []*pooledClient
	free    chan *pooledClient
}

type pooledClient struct {
	mu     sync.Mutex
	client *gosseract.Client
}

// NewTesseractPool builds size client instances, all configured with
// the given language list (typically ["eng", "ron"]).
//
// Returns an error if any client fails to construct — typically a
// missing tessdata file. We do not log-and-continue here: a partially
// constructed pool is worse than a fast failure at startup.
func NewTesseractPool(size int, langs []string) (*TesseractPool, error) {
	if size < 1 {
		return nil, fmt.Errorf("pool size must be >= 1, got %d", size)
	}
	pool := &TesseractPool{
		clients: make([]*pooledClient, 0, size),
		free:    make(chan *pooledClient, size),
	}
	for i := 0; i < size; i++ {
		c := gosseract.NewClient()
		if err := c.SetLanguage(langs...); err != nil {
			pool.Close()
			c.Close()
			return nil, fmt.Errorf("set language: %w", err)
		}
		pc := &pooledClient{client: c}
		pool.clients = append(pool.clients, pc)
		pool.free <- pc
	}
	return pool, nil
}

// Recognize implements OCREngine.
func (p *TesseractPool) Recognize(image []byte) (*OCROutput, error) {
	pc := <-p.free
	defer func() { p.free <- pc }()

	pc.mu.Lock()
	defer pc.mu.Unlock()

	if err := pc.client.SetImageFromBytes(image); err != nil {
		return nil, fmt.Errorf("%w: set image: %v", ErrTesseract, err)
	}
	text, err := pc.client.Text()
	if err != nil {
		return nil, fmt.Errorf("%w: text: %v", ErrTesseract, err)
	}
	boxes, err := pc.client.GetBoundingBoxes(gosseract.RIL_WORD)
	if err != nil {
		return nil, fmt.Errorf("%w: bounding boxes: %v", ErrTesseract, err)
	}

	out := &OCROutput{Text: text, Words: make([]Word, 0, len(boxes))}
	for _, b := range boxes {
		bb := b.Box
		out.Words = append(out.Words, Word{
			Text:       b.Word,
			Confidence: b.Confidence / 100.0,
			BBox: BBox{
				float64(bb.Min.X),
				float64(bb.Min.Y),
				float64(bb.Dx()),
				float64(bb.Dy()),
			},
		})
	}
	return out, nil
}

// Close releases all underlying gosseract clients.
func (p *TesseractPool) Close() error {
	if p == nil {
		return nil
	}
	for _, pc := range p.clients {
		if pc.client != nil {
			pc.client.Close()
		}
	}
	return nil
}
