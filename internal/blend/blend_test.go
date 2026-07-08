package blend

import (
	"testing"

	"github.com/fourhundredfour/pinky/internal/config"
)

// bgra builds a single-pixel BGRA buffer.
func bgra(b, g, r, a byte) []byte {
	return []byte{b, g, r, a}
}

func TestApplyZeroOpacityIsNoop(t *testing.T) {
	pix := bgra(10, 20, 30, 255)
	orig := append([]byte(nil), pix...)
	Apply(pix, Params{Mode: config.ModeTint, Color: config.RGB{R: 255}, Opacity: 0})
	for i := range pix {
		if pix[i] != orig[i] {
			t.Fatalf("Apply with opacity 0 mutated pixel: got %v, want %v", pix, orig)
		}
	}
}

func TestApplyTintFullOpacityReplacesWithColor(t *testing.T) {
	pix := bgra(10, 20, 30, 128) // arbitrary source color + alpha
	Apply(pix, Params{Mode: config.ModeTint, Color: config.RGB{R: 0xFF, G: 0x33, B: 0xAA}, Opacity: 1})

	wantB, wantG, wantR := byte(0xAA), byte(0x33), byte(0xFF)
	if pix[0] != wantB || pix[1] != wantG || pix[2] != wantR {
		t.Fatalf("got BGR=(%d,%d,%d), want (%d,%d,%d)", pix[0], pix[1], pix[2], wantB, wantG, wantR)
	}
	if pix[3] != 255 {
		t.Fatalf("output alpha = %d, want 255 (overlay must be fully opaque)", pix[3])
	}
}

func TestApplyTintHalfOpacityIsMidpoint(t *testing.T) {
	// Pure black source, pure white layer color, 50% opacity -> mid gray.
	pix := bgra(0, 0, 0, 255)
	Apply(pix, Params{Mode: config.ModeTint, Color: config.RGB{R: 255, G: 255, B: 255}, Opacity: 0.5})

	for i, name := range []string{"B", "G", "R"} {
		v := pix[i]
		if v < 126 || v > 129 {
			t.Errorf("%s channel = %d, want ~127-128 (midpoint of 0 and 255)", name, v)
		}
	}
}

func TestApplyMonochromeGrayscaleTimesColor(t *testing.T) {
	// Mid-gray source, red layer color, full opacity -> should stay on the
	// red axis with no green/blue contribution.
	pix := bgra(128, 128, 128, 255)
	Apply(pix, Params{Mode: config.ModeMonochrome, Color: config.RGB{R: 255, G: 0, B: 0}, Opacity: 1})

	if pix[0] != 0 || pix[1] != 0 { // B, G channels
		t.Errorf("expected B/G channels to be 0 for a pure-red monochrome tint, got B=%d G=%d", pix[0], pix[1])
	}
	if pix[2] == 0 {
		t.Errorf("expected R channel > 0 for a mid-gray source, got R=%d", pix[2])
	}
}

func TestApplyMultiplyBlackStaysBlack(t *testing.T) {
	pix := bgra(0, 0, 0, 255)
	Apply(pix, Params{Mode: config.ModeMultiply, Color: config.RGB{R: 255, G: 200, B: 100}, Opacity: 1})
	if pix[0] != 0 || pix[1] != 0 || pix[2] != 0 {
		t.Fatalf("multiply of black should stay black, got %v", pix[:3])
	}
}

func TestApplyMultiplyWhiteBecomesColor(t *testing.T) {
	pix := bgra(255, 255, 255, 255)
	Apply(pix, Params{Mode: config.ModeMultiply, Color: config.RGB{R: 255, G: 51, B: 170}, Opacity: 1})
	if pix[2] != 255 || pix[1] != 51 || pix[0] != 170 {
		t.Fatalf("multiply of white should equal the layer color, got BGR=%v want (170,51,255)", pix[:3])
	}
}

func TestApplyColorModeKeepsLuminance(t *testing.T) {
	// A bright source pixel run through the "color" blend should stay
	// bright even though its hue changes to the (dark) layer color.
	pix := bgra(230, 230, 230, 255) // light gray source
	Apply(pix, Params{Mode: config.ModeColor, Color: config.RGB{R: 50, G: 0, B: 50}, Opacity: 1})

	sum := int(pix[0]) + int(pix[1]) + int(pix[2])
	if sum < 300 {
		t.Fatalf("expected the result to remain bright (preserve source luminance), got BGR=%v (sum=%d)", pix[:3], sum)
	}
}

func TestApplyAlwaysForcesOpaqueAlpha(t *testing.T) {
	for _, mode := range []config.Mode{config.ModeTint, config.ModeMultiply, config.ModeMonochrome, config.ModeColor} {
		pix := bgra(1, 2, 3, 0)
		Apply(pix, Params{Mode: mode, Color: config.RGB{R: 100, G: 100, B: 100}, Opacity: 0.3})
		if pix[3] != 255 {
			t.Errorf("mode %s: alpha = %d, want 255", mode, pix[3])
		}
	}
}
