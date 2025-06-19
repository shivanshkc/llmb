package miscutils

import (
	"fmt"
	"time"
)

// FormatDuration ...
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	// Format based on magnitude.
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%.0fns", float64(d.Nanoseconds()))
	case d < time.Millisecond:
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000)
	case d < time.Second:
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000)
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}
