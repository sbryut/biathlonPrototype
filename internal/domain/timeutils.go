package domain

import (
	"fmt"
	"strings"
	"time"
)

const TimeLayout = "15:04:05.000"

// ParseTimeFromString parses time from a string of the format [HH:MM:SS.sss]
func ParseTimeFromString(timeStr string) (time.Time, error) {
	trimmedTimeStr := strings.Trim(timeStr, "[]")
	t, err := time.Parse(TimeLayout, trimmedTimeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse time '%s': %v", timeStr, err)
	}
	return t, nil
}

// ParseDurationFromString parses duration from a string HH:MM:SS or HH:MM:SS.sss
func ParseDurationFromString(durationStr string) (time.Duration, error) {
	parts := strings.Split(durationStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid duration format: %s", durationStr)
	}

	parseableStr := fmt.Sprintf("%sh%sm%ss", parts[0], parts[1], parts[2])
	dur, err := time.ParseDuration(parseableStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration '%s': %v", durationStr, err)
	}
	return dur, nil
}

// FormatTime parses time into a string of format [HH:MM:SS.sss]
func FormatTime(t time.Time) string {
	return fmt.Sprintf("[%s]", t.Format(TimeLayout))
}

// FormatDuration parses duration into a string of format HH:MM:SS.sss
func FormatDuration(dur time.Duration) string {
	dur = dur.Round(time.Millisecond)

	h := dur / time.Hour
	dur -= h * time.Hour

	m := dur / time.Minute
	dur -= m * time.Minute

	s := dur / time.Second
	dur -= s * time.Second

	ms := dur / time.Millisecond

	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

// CalculateSpeed calculates the speed (m/s)
func CalculateSpeed(distance float64, duration time.Duration) float64 {
	if duration <= 0 {
		return 0.0
	}
	return distance / duration.Seconds()
}
