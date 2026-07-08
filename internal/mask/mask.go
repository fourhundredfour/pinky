// Package mask computes a per-pixel "content" alpha for a captured taskbar
// strip: high where an icon/text glyph is, zero where it is background. The
// blend stage multiplies the layer color by this alpha, so only glyph pixels
// get colored and the real taskbar/wallpaper shows through everywhere else.
//
// It is intentionally OS-independent (operates on a raw BGRA byte slice plus
// dimensions and plain integer rectangles) so it can be unit-tested on any
// platform, like the blend package.
package mask

import "sort"

// Rect is a plain integer rectangle in strip-local pixel coordinates
// (0,0 = top-left of the captured strip). It mirrors a win32.RECT but avoids
// importing the Windows-only win32 package here.
type Rect struct {
	Left, Top, Right, Bottom int
}

func (r Rect) width() int  { return r.Right - r.Left }
func (r Rect) height() int { return r.Bottom - r.Top }

// Compute returns one alpha byte per pixel (row-major, width*height bytes).
//
// When cells is non-nil, alpha is non-zero ONLY inside those cells, and
// within each cell a local background-subtraction isolates the glyph from
// the cell's padding. A non-nil but empty cells slice therefore yields a
// fully transparent mask (UI Automation ran but nothing was selected).
//
// When cells is nil (e.g. UI Automation was unavailable), it falls back to a
// whole-strip background subtraction so the effect still works, just less
// precisely.
//
// sensitivity in [0,1] trades off inclusiveness: higher pulls in fainter
// pixels (more of the glyph, but more risk of background noise).
func Compute(pix []byte, width, height int, cells []Rect, sensitivity float64) []byte {
	alpha := make([]byte, width*height)
	if width <= 0 || height <= 0 || len(pix) < width*height*4 {
		return alpha
	}

	lo, hi := thresholds(sensitivity)

	if cells == nil {
		computeFullStrip(pix, width, height, lo, hi, alpha)
		return alpha
	}

	for _, c := range cells {
		computeCell(pix, width, height, clip(c, width, height), lo, hi, alpha)
	}
	return alpha
}

// thresholds maps sensitivity to the low/high deviation cut-offs of the
// smoothstep. More sensitive => lower thresholds => more pixels count as
// content.
func thresholds(sensitivity float64) (lo, hi float64) {
	s := clampF(sensitivity, 0, 1)
	lo = lerp(0.25, 0.03, s)
	hi = lerp(0.50, 0.12, s)
	return lo, hi
}

func computeCell(pix []byte, width, height int, c Rect, lo, hi float64, alpha []byte) {
	if c.width() <= 0 || c.height() <= 0 {
		return
	}
	br, bg, bb := cellBackground(pix, width, c)
	for y := c.Top; y < c.Bottom; y++ {
		row := y * width
		for x := c.Left; x < c.Right; x++ {
			o := (row + x) * 4
			d := deviation(pix[o+2], pix[o+1], pix[o], br, bg, bb)
			alpha[row+x] = toByte(smoothstep(lo, hi, d))
		}
	}
}

// cellBackground estimates the cell's background color as the per-channel
// median of its border pixels. A taskbar button is a glyph on a small,
// mostly-uniform pad, so its border is almost always background.
func cellBackground(pix []byte, width int, c Rect) (r, g, b float64) {
	var rs, gs, bs []float64
	sample := func(x, y int) {
		o := (y*width + x) * 4
		bs = append(bs, float64(pix[o]))
		gs = append(gs, float64(pix[o+1]))
		rs = append(rs, float64(pix[o+2]))
	}
	for x := c.Left; x < c.Right; x++ {
		sample(x, c.Top)
		sample(x, c.Bottom-1)
	}
	for y := c.Top; y < c.Bottom; y++ {
		sample(c.Left, y)
		sample(c.Right-1, y)
	}
	return median(rs) / 255, median(gs) / 255, median(bs) / 255
}

// computeFullStrip is the fallback path: estimate a slowly-varying
// background via a wide box blur (radius ~ the bar thickness) using an
// integral image, then mark pixels that deviate from it.
func computeFullStrip(pix []byte, width, height int, lo, hi float64, alpha []byte) {
	radius := height
	if radius < 8 {
		radius = 8
	}
	sr := integral(pix, width, height, 2)
	sg := integral(pix, width, height, 1)
	sb := integral(pix, width, height, 0)

	for y := 0; y < height; y++ {
		row := y * width
		for x := 0; x < width; x++ {
			br := boxAvg(sr, width, height, x, y, radius) / 255
			bg := boxAvg(sg, width, height, x, y, radius) / 255
			bb := boxAvg(sb, width, height, x, y, radius) / 255
			o := (row + x) * 4
			d := deviation(pix[o+2], pix[o+1], pix[o], br, bg, bb)
			alpha[row+x] = toByte(smoothstep(lo, hi, d))
		}
	}
}

// deviation measures how far a pixel is from the background color, combining
// luminance and chroma so both bright/dark glyphs and colorful glyphs on a
// similar-luminance background are detected. Result is roughly in [0,1].
func deviation(r8, g8, b8 byte, br, bg, bb float64) float64 {
	r := float64(r8) / 255
	g := float64(g8) / 255
	b := float64(b8) / 255
	dl := abs(luminance(r, g, b) - luminance(br, bg, bb))
	dc := (abs(r-br) + abs(g-bg) + abs(b-bb)) / 3
	return maxF(dl, dc)
}

func luminance(r, g, b float64) float64 { return 0.299*r + 0.587*g + 0.114*b }

// integral builds a (width+1)x(height+1) summed-area table for one BGRA
// channel (0=B, 1=G, 2=R).
func integral(pix []byte, width, height, channel int) []float64 {
	stride := width + 1
	sum := make([]float64, stride*(height+1))
	for y := 0; y < height; y++ {
		rowAbove := y * stride
		rowCur := (y + 1) * stride
		var lineSum float64
		for x := 0; x < width; x++ {
			lineSum += float64(pix[(y*width+x)*4+channel])
			sum[rowCur+x+1] = sum[rowAbove+x+1] + lineSum
		}
	}
	return sum
}

// boxAvg returns the average channel value in the square window of the given
// radius around (x,y), using the summed-area table.
func boxAvg(sum []float64, width, height, x, y, radius int) float64 {
	x0 := x - radius
	y0 := y - radius
	x1 := x + radius + 1
	y1 := y + radius + 1
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > width {
		x1 = width
	}
	if y1 > height {
		y1 = height
	}
	stride := width + 1
	total := sum[y1*stride+x1] - sum[y0*stride+x1] - sum[y1*stride+x0] + sum[y0*stride+x0]
	area := float64((x1 - x0) * (y1 - y0))
	if area == 0 {
		return 0
	}
	return total / area
}

func clip(r Rect, width, height int) Rect {
	if r.Left < 0 {
		r.Left = 0
	}
	if r.Top < 0 {
		r.Top = 0
	}
	if r.Right > width {
		r.Right = width
	}
	if r.Bottom > height {
		r.Bottom = height
	}
	return r
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sort.Float64s(v)
	n := len(v)
	if n%2 == 1 {
		return v[n/2]
	}
	return (v[n/2-1] + v[n/2]) / 2
}

func smoothstep(lo, hi, x float64) float64 {
	if hi <= lo {
		if x >= hi {
			return 1
		}
		return 0
	}
	t := clampF((x-lo)/(hi-lo), 0, 1)
	return t * t * (3 - 2*t)
}

func toByte(v float64) byte {
	v *= 255
	switch {
	case v <= 0:
		return 0
	case v >= 255:
		return 255
	default:
		return byte(v + 0.5)
	}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
