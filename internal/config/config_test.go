package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	def := Default()
	if cfg.Edge != def.Edge || cfg.Size != def.Size || cfg.Alignment != def.Alignment {
		t.Fatalf("Load did not return defaults: %+v", cfg)
	}

	// The file should now exist and round-trip through Load again.
	cfg2, err := Load(path)
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if cfg2.BackgroundColor != cfg.BackgroundColor {
		t.Fatalf("round-tripped config differs: %+v vs %+v", cfg, cfg2)
	}
}

func TestLoadAppliesDefaultsForPartialFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := writeFile(path, "edge = \"left\"\n"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Edge != EdgeLeft {
		t.Fatalf("Edge = %q, want left", cfg.Edge)
	}
	if cfg.Size != Default().Size {
		t.Fatalf("Size = %d, want default %d", cfg.Size, Default().Size)
	}
	if cfg.ClockFormat != Default().ClockFormat {
		t.Fatalf("ClockFormat = %q, want default", cfg.ClockFormat)
	}
}

func TestLoadRejectsInvalidEdge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := writeFile(path, "edge = \"diagonal\"\n"); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected error for invalid edge, got nil")
	}
}

func TestApplyBoundsAndDefaultsClampsOpacity(t *testing.T) {
	cfg := Default()
	cfg.BackgroundOpacity = 5
	cfg.applyBoundsAndDefaults()
	if cfg.BackgroundOpacity != 1 {
		t.Fatalf("BackgroundOpacity = %v, want clamped to 1", cfg.BackgroundOpacity)
	}

	cfg.BackgroundOpacity = -3
	cfg.applyBoundsAndDefaults()
	if cfg.BackgroundOpacity != 0 {
		t.Fatalf("BackgroundOpacity = %v, want clamped to 0", cfg.BackgroundOpacity)
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		in      string
		want    RGB
		wantErr bool
	}{
		{"#FF33AA", RGB{0xFF, 0x33, 0xAA}, false},
		{"3AA0FF", RGB{0x3A, 0xA0, 0xFF}, false},
		{"#zzzzzz", RGB{}, true},
		{"#FFF", RGB{}, true},
	}
	for _, tt := range tests {
		got, err := ParseColor(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseColor(%q): expected error, got %+v", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseColor(%q): unexpected error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseColor(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestEdgeHorizontal(t *testing.T) {
	if !EdgeTop.Horizontal() || !EdgeBottom.Horizontal() {
		t.Error("top/bottom should be horizontal")
	}
	if EdgeLeft.Horizontal() || EdgeRight.Horizontal() {
		t.Error("left/right should not be horizontal")
	}
}

func writeFile(path, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o644)
}
