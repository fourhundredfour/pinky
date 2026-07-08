package tasks

import "testing"

func TestShouldList(t *testing.T) {
	tests := []struct {
		name string
		c    candidate
		want bool
	}{
		{
			name: "normal visible titled window",
			c:    candidate{Title: "Notepad", Visible: true},
			want: true,
		},
		{
			name: "invisible window",
			c:    candidate{Title: "Notepad", Visible: false},
			want: false,
		},
		{
			name: "cloaked window (suspended UWP)",
			c:    candidate{Title: "Notepad", Visible: true, Cloaked: true},
			want: false,
		},
		{
			name: "no title",
			c:    candidate{Title: "", Visible: true},
			want: false,
		},
		{
			name: "owned utility window without APPWINDOW flag",
			c:    candidate{Title: "Find", Visible: true, HasOwner: true},
			want: false,
		},
		{
			name: "tool window",
			c:    candidate{Title: "Toolbox", Visible: true, ExStyle: exStyleToolWindow},
			want: false,
		},
		{
			name: "owned window explicitly forced onto the taskbar",
			c:    candidate{Title: "Picture-in-Picture", Visible: true, HasOwner: true, ExStyle: exStyleAppWindow},
			want: true,
		},
		{
			name: "tool window explicitly forced onto the taskbar",
			c:    candidate{Title: "Palette", Visible: true, ExStyle: exStyleToolWindow | exStyleAppWindow},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldList(tt.c); got != tt.want {
				t.Errorf("shouldList(%+v) = %v, want %v", tt.c, got, tt.want)
			}
		})
	}
}
