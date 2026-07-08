// Package clock renders the current time/date using configurable Go
// time-layout strings and pushes the result to the frontend on an
// interval. The rendering logic has no OS dependency so it can be
// unit-tested anywhere.
package clock

import "time"

// Tick is the payload pushed to the frontend on every tick.
type Tick struct {
	Time string `json:"time"`
	Date string `json:"date"`
}

// Format holds the two Go time-layout strings (e.g. "15:04", "03:04 PM")
// used to render a Tick.
type Format struct {
	Time string
	Date string
}

// DefaultFormat matches config.Default()'s clock/date formats.
var DefaultFormat = Format{Time: "15:04", Date: "Monday, 02 January 2006"}

// Render formats now using f, falling back to DefaultFormat for any blank
// field.
func Render(now time.Time, f Format) Tick {
	timeLayout := f.Time
	if timeLayout == "" {
		timeLayout = DefaultFormat.Time
	}
	dateLayout := f.Date
	if dateLayout == "" {
		dateLayout = DefaultFormat.Date
	}
	return Tick{
		Time: now.Format(timeLayout),
		Date: now.Format(dateLayout),
	}
}
