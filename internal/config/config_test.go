package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCreatesDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	def := Default()
	if *cfg != *def {
		t.Fatalf("expected defaults %+v, got %+v", def, cfg)
	}
}

func TestLoadAppliesBoundsForPartialFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := writeFile(path, "color: \"#00FF00\"\n"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Color != "#00FF00" {
		t.Errorf("Color = %q, want #00FF00", cfg.Color)
	}
	if cfg.Mode != Default().Mode {
		t.Errorf("Mode = %q, want default %q", cfg.Mode, Default().Mode)
	}
	if cfg.FPS != Default().FPS {
		t.Errorf("FPS = %d, want default %d", cfg.FPS, Default().FPS)
	}
}

func TestLoadRejectsUnknownMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := writeFile(path, "mode: rainbow\n"); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected error for unknown mode, got nil")
	}
}

func TestLoadRejectsBadColor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := writeFile(path, "color: not-a-color\n"); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected error for invalid color, got nil")
	}
}

func TestParseColor(t *testing.T) {
	cases := []struct {
		in      string
		want    RGB
		wantErr bool
	}{
		{"#FF33AA", RGB{0xFF, 0x33, 0xAA}, false},
		{"ff33aa", RGB{0xFF, 0x33, 0xAA}, false},
		{"#000000", RGB{0, 0, 0}, false},
		{"#zzzzzz", RGB{}, true},
		{"#fff", RGB{}, true},
	}
	for _, c := range cases {
		got, err := ParseColor(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("ParseColor(%q) error = %v, wantErr %v", c.in, err, c.wantErr)
			continue
		}
		if err == nil && got != c.want {
			t.Errorf("ParseColor(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestWatcherDetectsChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := writeFile(path, "opacity: 0.5\n"); err != nil {
		t.Fatal(err)
	}

	w := NewWatcher(path)
	cfg, changed, err := w.Poll()
	if err != nil {
		t.Fatalf("first Poll: %v", err)
	}
	if !changed || cfg.Opacity != 0.5 {
		t.Fatalf("first Poll: changed=%v cfg=%+v, want changed with opacity 0.5", changed, cfg)
	}

	_, changed, err = w.Poll()
	if err != nil {
		t.Fatalf("second Poll: %v", err)
	}
	if changed {
		t.Fatal("second Poll reported changed with no file modification")
	}

	// Ensure the mtime strictly advances on filesystems with coarse
	// resolution before rewriting the file.
	time.Sleep(10 * time.Millisecond)
	if err := writeFile(path, "opacity: 0.9\n"); err != nil {
		t.Fatal(err)
	}

	cfg, changed, err = w.Poll()
	if err != nil {
		t.Fatalf("third Poll: %v", err)
	}
	if !changed || cfg.Opacity != 0.9 {
		t.Fatalf("third Poll: changed=%v cfg=%+v, want changed with opacity 0.9", changed, cfg)
	}
}

func writeFile(path, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o644)
}
