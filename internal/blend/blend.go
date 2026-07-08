// Package blend implements the graphic-design "layer" effects (monochrome,
// tint, multiply, color) applied to a captured taskbar frame before it is
// drawn back as the overlay.
package blend

import (
	"math"

	"github.com/fourhundredfour/pinky/internal/config"
)

// Params configures a single Apply call.
type Params struct {
	Mode    config.Mode
	Color   config.RGB
	Opacity float64
}

// Apply transforms pix (a BGRA buffer such as capture.Frame.Pix) in place
// according to p, using mask as a per-pixel content alpha (one byte per
// pixel, same pixel order as pix). Pixels whose mask is 0 become fully
// transparent so the real taskbar/wallpaper shows through untouched; only
// masked-in glyph pixels get colored.
//
// Each mode first computes a "blend result" color from the source pixel and
// the configured layer color, then that is composited as
// lerp(source, blendResult, Opacity) - exactly how a Photoshop layer with a
// blend mode and an opacity slider composites against the layer below it.
// The layer color is composited against the just-captured source here (in
// software) rather than left for the OS compositor to blend against the
// live/still-animating real taskbar, which would otherwise ghost between
// capture ticks.
//
// Output is written as PREMULTIPLIED BGRA because the overlay draws with
// AC_SRC_ALPHA: the final per-pixel alpha is (mask/255 * Opacity), and the
// color channels are multiplied by that alpha. When mask is nil, every pixel
// is treated as fully masked-in (alpha driven solely by Opacity).
func Apply(pix []byte, mask []byte, p Params) {
	if p.Opacity <= 0 {
		for i := 0; i+3 < len(pix); i += 4 {
			pix[i], pix[i+1], pix[i+2], pix[i+3] = 0, 0, 0, 0
		}
		return
	}
	opacity := p.Opacity
	if opacity > 1 {
		opacity = 1
	}

	cr := float64(p.Color.R) / 255
	cg := float64(p.Color.G) / 255
	cb := float64(p.Color.B) / 255

	for i, px := 0, 0; i+3 < len(pix); i, px = i+4, px+1 {
		coverage := 1.0
		if mask != nil {
			if px >= len(mask) {
				break
			}
			coverage = float64(mask[px]) / 255
		}
		a := coverage * opacity
		if a <= 0 {
			pix[i], pix[i+1], pix[i+2], pix[i+3] = 0, 0, 0, 0
			continue
		}

		b := float64(pix[i]) / 255
		g := float64(pix[i+1]) / 255
		r := float64(pix[i+2]) / 255

		var rr, gg, bb float64
		switch p.Mode {
		case config.ModeTint:
			rr, gg, bb = cr, cg, cb
		case config.ModeMultiply:
			rr, gg, bb = r*cr, g*cg, b*cb
		case config.ModeMonochrome:
			l := luminance(r, g, b)
			rr, gg, bb = l*cr, l*cg, l*cb
		case config.ModeColor:
			rr, gg, bb = colorBlend(r, g, b, cr, cg, cb)
		default:
			rr, gg, bb = r, g, b
		}

		// Composite the layer over the source, then premultiply by alpha.
		outR := lerp(r, rr, opacity)
		outG := lerp(g, gg, opacity)
		outB := lerp(b, bb, opacity)
		pix[i] = clampByte(outB * a)
		pix[i+1] = clampByte(outG * a)
		pix[i+2] = clampByte(outR * a)
		pix[i+3] = clampByte(a)
	}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func clampByte(v float64) byte {
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

// luminance uses the standard PDF/Photoshop non-separable blend mode
// luminosity weights.
func luminance(r, g, b float64) float64 {
	return 0.3*r + 0.59*g + 0.11*b
}

// colorBlend reproduces Photoshop's "Color" blend mode: the result keeps
// the base pixel's luminosity but takes hue and saturation from the layer
// color. This is the standard PDF spec non-separable blend algorithm
// (Lum / ClipColor / SetLum).
func colorBlend(baseR, baseG, baseB, colR, colG, colB float64) (float64, float64, float64) {
	targetLum := luminance(baseR, baseG, baseB)
	r, g, b := setLum(colR, colG, colB, targetLum)
	return r, g, b
}

func setLum(r, g, b, l float64) (float64, float64, float64) {
	d := l - luminance(r, g, b)
	return clipColor(r+d, g+d, b+d)
}

func clipColor(r, g, b float64) (float64, float64, float64) {
	l := luminance(r, g, b)
	n := math.Min(r, math.Min(g, b))
	x := math.Max(r, math.Max(g, b))

	if n < 0 && l != n {
		r = l + (r-l)*l/(l-n)
		g = l + (g-l)*l/(l-n)
		b = l + (b-l)*l/(l-n)
	}
	if x > 1 && x != l {
		r = l + (r-l)*(1-l)/(x-l)
		g = l + (g-l)*(1-l)/(x-l)
		b = l + (b-l)*(1-l)/(x-l)
	}
	return r, g, b
}
