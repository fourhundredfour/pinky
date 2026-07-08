package clock

import (
	"testing"
	"time"
)

func TestRender(t *testing.T) {
	now := time.Date(2026, time.July, 9, 14, 5, 0, 0, time.UTC)

	got := Render(now, Format{Time: "15:04", Date: "2006-01-02"})
	want := Tick{Time: "14:05", Date: "2026-07-09"}
	if got != want {
		t.Errorf("Render() = %+v, want %+v", got, want)
	}
}

func TestRenderFallsBackToDefaultsForBlankFields(t *testing.T) {
	now := time.Date(2026, time.July, 9, 14, 5, 0, 0, time.UTC)

	got := Render(now, Format{})
	want := Render(now, DefaultFormat)
	if got != want {
		t.Errorf("Render() with blank format = %+v, want default %+v", got, want)
	}
}

func Test12HourFormat(t *testing.T) {
	now := time.Date(2026, time.July, 9, 14, 5, 0, 0, time.UTC)

	got := Render(now, Format{Time: "03:04 PM"})
	if got.Time != "02:05 PM" {
		t.Errorf("Time = %q, want %q", got.Time, "02:05 PM")
	}
}
