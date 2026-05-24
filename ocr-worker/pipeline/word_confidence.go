package pipeline

import "strings"

// wordConfidence aggregates per-word confidence into per-substring
// confidence. Given the OCR's word list, you ask "what's the confidence
// of this substring of the full text?" — we find every word whose
// surface form appears in the substring and aggregate.
//
// We use the MIN of contributing word confidences (not the average).
// One garbled word makes the whole field unreliable, and we'd rather
// flag a field for review than persist a corrupted value.
type wordConfidence struct {
	words    []Word
	fallback float64
}

func newWordConfidence(words []Word, fallback float64) *wordConfidence {
	if fallback < 0 {
		fallback = 0
	}
	if fallback > 1 {
		fallback = 1
	}
	return &wordConfidence{words: words, fallback: fallback}
}

// scoreSubstring returns (confidence, union-bbox) for the given text
// substring. Confidence is the min of all OCR words whose surface form
// appears as a token in the substring. Bbox is the smallest box
// containing all contributing words.
//
// If the engine returned no word data, returns (fallback, nil).
func (w *wordConfidence) scoreSubstring(s string) (float64, *BBox) {
	if len(w.words) == 0 || s == "" {
		return w.fallback, nil
	}
	tokens := tokenize(s)
	if len(tokens) == 0 {
		return w.fallback, nil
	}
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, tk := range tokens {
		tokenSet[strings.ToLower(tk)] = struct{}{}
	}

	min := -1.0
	var minX, minY, maxR, maxB float64
	have := false
	for _, word := range w.words {
		wt := strings.ToLower(strings.TrimSpace(word.Text))
		if wt == "" {
			continue
		}
		if _, ok := tokenSet[wt]; !ok {
			continue
		}
		if min < 0 || word.Confidence < min {
			min = word.Confidence
		}
		x, y, ww, hh := word.BBox[0], word.BBox[1], word.BBox[2], word.BBox[3]
		if !have {
			minX, minY = x, y
			maxR, maxB = x+ww, y+hh
			have = true
		} else {
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x+ww > maxR {
				maxR = x + ww
			}
			if y+hh > maxB {
				maxB = y + hh
			}
		}
	}
	if min < 0 {
		return w.fallback, nil
	}
	bbox := BBox{minX, minY, maxR - minX, maxB - minY}
	return min, &bbox
}

// tokenize splits on whitespace and punctuation. We intentionally don't
// import strings.FieldsFunc-with-unicode because the OCR text already
// went through Tesseract's tokenizer; this is a coarse second pass.
func tokenize(s string) []string {
	out := make([]string, 0, 4)
	var cur strings.Builder
	for _, r := range s {
		if isWordRune(r) {
			cur.WriteRune(r)
		} else if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func isWordRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	}
	// Romanian diacritics.
	switch r {
	case 'ă', 'â', 'î', 'ș', 'ț', 'Ă', 'Â', 'Î', 'Ș', 'Ț':
		return true
	}
	return false
}
