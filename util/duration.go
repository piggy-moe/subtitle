package util

import (
	"fmt"
	"time"
)

func ParseDuration(s string) (time.Duration, error) {
	var hh, mm, ss int
	if _, err := fmt.Sscanf(s, "%02d:%02d:%02d", &hh, &mm, &ss); err != nil {
		return 0, fmt.Errorf("unable to parse %s: %s", s, err.Error())
	}
	return time.Duration(hh)*time.Hour + time.Duration(mm)*time.Minute + time.Duration(ss)*time.Second, nil
}

func FormatDuration(t time.Duration) string {
	return fmt.Sprintf("%02d:%02d:%02d", int(t.Hours()), int(t.Minutes())%60, int(t.Seconds())%60)
}
