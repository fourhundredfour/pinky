package mask

import "testing"

// synthetic builds a width x height BGRA strip filled with a uniform
// background color; the returned buffer can then be drawn onto.
func synthetic(width, height int, bg [3]byte) []byte {
	pix := make([]byte, width*height*4)
	for i := 0; i < width*height; i++ {
		o := i * 4
		pix[o] = bg[2]   // B
		pix[o+1] = bg[1] // G
		pix[o+2] = bg[0] // R
		pix[o+3] = 255
	}
	return pix
}

func setPixel(pix []byte, width, x, y int, c [3]byte) {
	o := (y*width + x) * 4
	pix[o] = c[2]
	pix[o+1] = c[1]
	pix[o+2] = c[0]
	pix[o+3] = 255
}

func alphaAt(alpha []byte, width, x, y int) byte { return alpha[y*width+x] }

func TestComputeIsZeroOutsideCells(t *testing.T) {
	w, h := 40, 10
	pix := synthetic(w, h, [3]byte{20, 20, 20})
	// Draw a bright block across the whole strip; only a cell in the middle
	// should ever be colored.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			setPixel(pix, w, x, y, [3]byte{240, 240, 240})
		}
	}

	cell := Rect{Left: 15, Top: 2, Right: 25, Bottom: 8}
	alpha := Compute(pix, w, h, []Rect{cell}, 0.5)

	// Everything outside the cell must be zero.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			inside := x >= cell.Left && x < cell.Right && y >= cell.Top && y < cell.Bottom
			if !inside && alphaAt(alpha, w, x, y) != 0 {
				t.Fatalf("pixel (%d,%d) outside cell has alpha %d, want 0", x, y, alphaAt(alpha, w, x, y))
			}
		}
	}
}

func TestComputeGlyphInsideCellGetsAlpha(t *testing.T) {
	w, h := 30, 20
	// Uniform dark background (this is the cell's "pad").
	pix := synthetic(w, h, [3]byte{25, 25, 25})

	cell := Rect{Left: 5, Top: 3, Right: 25, Bottom: 17}
	// A bright glyph pixel well inside the cell (away from the border used
	// for background estimation).
	gx, gy := 15, 10
	setPixel(pix, w, gx, gy, [3]byte{250, 250, 250})

	alpha := Compute(pix, w, h, []Rect{cell}, 0.5)

	if got := alphaAt(alpha, w, gx, gy); got < 200 {
		t.Fatalf("bright glyph pixel alpha = %d, want high (>=200)", got)
	}
	// A pad pixel matching the background should stay near-zero.
	if got := alphaAt(alpha, w, cell.Left+1, cell.Top+1); got > 20 {
		t.Fatalf("uniform pad pixel alpha = %d, want ~0", got)
	}
}

func TestComputeEmptyCellsIsFullyTransparent(t *testing.T) {
	w, h := 20, 10
	pix := synthetic(w, h, [3]byte{200, 100, 50})
	// Non-nil but empty: UIA ran, selected nothing -> everything transparent.
	alpha := Compute(pix, w, h, []Rect{}, 0.5)
	for i, v := range alpha {
		if v != 0 {
			t.Fatalf("empty (non-nil) cells should be fully transparent, alpha[%d]=%d", i, v)
		}
	}
}

func TestComputeFallbackSeparatesBlockFromGradient(t *testing.T) {
	w, h := 60, 16
	pix := make([]byte, w*h*4)
	// Smooth horizontal gradient background.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := byte(40 + (x*120)/w)
			setPixel(pix, w, x, y, [3]byte{v, v, v})
		}
	}
	// A solid bright block on the left.
	block := Rect{Left: 5, Top: 4, Right: 15, Bottom: 12}
	for y := block.Top; y < block.Bottom; y++ {
		for x := block.Left; x < block.Right; x++ {
			setPixel(pix, w, x, y, [3]byte{255, 0, 255})
		}
	}

	// nil cells -> fallback whole-strip detection.
	alpha := Compute(pix, w, h, nil, 0.5)

	// The block center should have alpha; a far-away smooth-gradient pixel
	// should stay low.
	if got := alphaAt(alpha, w, 10, 8); got < 120 {
		t.Fatalf("fallback: block pixel alpha = %d, want substantial", got)
	}
	if got := alphaAt(alpha, w, 50, 8); got > 60 {
		t.Fatalf("fallback: smooth gradient pixel alpha = %d, want low", got)
	}
}
