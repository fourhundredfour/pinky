package blend

import (
	"testing"

	"github.com/fourhundredfour/pinky/internal/config"
)

// bgra builds a single-pixel BGRA buffer.
func bgra(b, g, r, a byte) []byte {
	return []byte{b, g, r, a}
}

// fullMask returns a mask that marks every pixel fully opaque.
func fullMask(n int) []byte {
	m := make([]byte, n)
	for i := range m {
		m[i] = 255
	}
	return m
}

func TestApplyZeroOpacityClearsToTransparent(t *testing.T) {
	pix := bgra(10, 20, 30, 255)
	Apply(pix, nil, Params{Mode: config.ModeTint, Color: config.RGB{R: 255}, Opacity: 0})
	for i, v := range pix {
		if v != 0 {
			t.Fatalf("opacity 0 should clear pixel to transparent, byte %d = %d", i, v)
		}
	}
}

func TestApplyMaskZeroIsTransparent(t *testing.T) {
	pix := bgra(10, 20, 30, 255)
	Apply(pix, []byte{0}, Params{Mode: config.ModeTint, Color: config.RGB{R: 255, G: 255, B: 255}, Opacity: 1})
	for i, v := range pix {
		if v != 0 {
			t.Fatalf("mask 0 should yield a transparent pixel, byte %d = %d", i, v)
		}
	}
}

func TestApplyNilMaskTintFullOpacity(t *testing.T) {
	// nil mask means "fully masked in"; at opacity 1 alpha is 255 and the
	// premultiplied color equals the layer color.
	pix := bgra(10, 20, 30, 128)
	Apply(pix, nil, Params{Mode: config.ModeTint, Color: config.RGB{R: 0xFF, G: 0x33, B: 0xAA}, Opacity: 1})

	wantB, wantG, wantR := byte(0xAA), byte(0x33), byte(0xFF)
	if pix[0] != wantB || pix[1] != wantG || pix[2] != wantR {
		t.Fatalf("got BGR=(%d,%d,%d), want (%d,%d,%d)", pix[0], pix[1], pix[2], wantB, wantG, wantR)
	}
	if pix[3] != 255 {
		t.Fatalf("output alpha = %d, want 255 at opacity 1 + full mask", pix[3])
	}
}

func TestApplyPremultipliedByOpacity(t *testing.T) {
	// White layer over black source, opacity 0.3, full mask.
	// Composited color = lerp(0,1,0.3) = 0.3; alpha = 0.3.
	// Premultiplied color channel = 0.3*0.3 = 0.09 -> ~23; alpha -> ~77.
	pix := bgra(0, 0, 0, 255)
	Apply(pix, fullMask(1), Params{Mode: config.ModeTint, Color: config.RGB{R: 255, G: 255, B: 255}, Opacity: 0.3})

	wantColor, wantAlpha := byte(23), byte(77)
	for i, name := range []string{"B", "G", "R"} {
		if diff := int(pix[i]) - int(wantColor); diff < -1 || diff > 1 {
			t.Errorf("%s = %d, want ~%d (color premultiplied by alpha)", name, pix[i], wantColor)
		}
	}
	if diff := int(pix[3]) - int(wantAlpha); diff < -1 || diff > 1 {
		t.Errorf("A = %d, want ~%d", pix[3], wantAlpha)
	}
	if pix[3] == 255 {
		t.Fatalf("alpha should not be forced opaque anymore; got 255")
	}
}

func TestApplyPremultipliedByMask(t *testing.T) {
	// White layer over black source, opacity 1, mask 128 -> alpha ~0.502,
	// premultiplied color equals alpha.
	pix := bgra(0, 0, 0, 255)
	Apply(pix, []byte{128}, Params{Mode: config.ModeTint, Color: config.RGB{R: 255, G: 255, B: 255}, Opacity: 1})

	want := byte(128)
	for i, name := range []string{"B", "G", "R", "A"} {
		if diff := int(pix[i]) - int(want); diff < -2 || diff > 2 {
			t.Errorf("%s = %d, want ~%d (scaled by mask coverage)", name, pix[i], want)
		}
	}
}

func TestApplyMonochromeGrayscaleTimesColor(t *testing.T) {
	// Mid-gray source, red layer color, full opacity + mask -> stays on the
	// red axis with no green/blue contribution (alpha 1 so premultiply is a
	// no-op on the color).
	pix := bgra(128, 128, 128, 255)
	Apply(pix, nil, Params{Mode: config.ModeMonochrome, Color: config.RGB{R: 255, G: 0, B: 0}, Opacity: 1})

	if pix[0] != 0 || pix[1] != 0 {
		t.Errorf("expected B/G channels to be 0 for a pure-red monochrome tint, got B=%d G=%d", pix[0], pix[1])
	}
	if pix[2] == 0 {
		t.Errorf("expected R channel > 0 for a mid-gray source, got R=%d", pix[2])
	}
	if pix[3] != 255 {
		t.Errorf("alpha = %d, want 255", pix[3])
	}
}

func TestApplyMultiplyBlackStaysBlack(t *testing.T) {
	pix := bgra(0, 0, 0, 255)
	Apply(pix, nil, Params{Mode: config.ModeMultiply, Color: config.RGB{R: 255, G: 200, B: 100}, Opacity: 1})
	if pix[0] != 0 || pix[1] != 0 || pix[2] != 0 {
		t.Fatalf("multiply of black should stay black, got %v", pix[:3])
	}
}

func TestApplyMultiplyWhiteBecomesColor(t *testing.T) {
	pix := bgra(255, 255, 255, 255)
	Apply(pix, nil, Params{Mode: config.ModeMultiply, Color: config.RGB{R: 255, G: 51, B: 170}, Opacity: 1})
	if pix[2] != 255 || pix[1] != 51 || pix[0] != 170 {
		t.Fatalf("multiply of white should equal the layer color, got BGR=%v want (170,51,255)", pix[:3])
	}
}

func TestApplyColorModeKeepsLuminance(t *testing.T) {
	// A bright source pixel run through the "color" blend should stay bright
	// (opacity 1 + full mask means alpha 1, so no darkening from premultiply).
	pix := bgra(230, 230, 230, 255)
	Apply(pix, nil, Params{Mode: config.ModeColor, Color: config.RGB{R: 50, G: 0, B: 50}, Opacity: 1})

	sum := int(pix[0]) + int(pix[1]) + int(pix[2])
	if sum < 300 {
		t.Fatalf("expected the result to remain bright (preserve source luminance), got BGR=%v (sum=%d)", pix[:3], sum)
	}
}
