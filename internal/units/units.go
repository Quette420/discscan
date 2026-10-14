// Package units parses and formats human-friendly sizes, durations and counts.
package units

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	KB int64 = 1 << 10
	MB int64 = 1 << 20
	GB int64 = 1 << 30
	TB int64 = 1 << 40
)

// FormatSize renders a byte count using binary units: 1536 -> "1.50 KB".
func FormatSize(n int64) string {
	if n < KB {
		return fmt.Sprintf("%d B", n)
	}

	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	v := float64(n) / float64(KB)
	i := 0

	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}

	return fmt.Sprintf("%.2f %s", v, units[i])
}

var sizeUnits = map[string]int64{
	"":  MB, // a bare number means megabytes
	"b": 1,
	"k": KB, "kb": KB, "kib": KB, "к": KB, "кб": KB,
	"m": MB, "mb": MB, "mib": MB, "м": MB, "мб": MB,
	"g": GB, "gb": GB, "gib": GB, "г": GB, "гб": GB,
	"t": TB, "tb": TB, "tib": TB, "т": TB, "тб": TB,
}

// ParseSize parses sizes like "500MB", "1.5 GB", "2g" or "200" (megabytes).
func ParseSize(s string) (int64, error) {
	num, unit := splitNumber(s)

	mult, ok := sizeUnits[unit]
	if !ok || num == "" {
		return 0, fmt.Errorf("invalid size %q (examples: 500MB, 1.5GB, 200)", s)
	}

	v, err := strconv.ParseFloat(num, 64)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("invalid size %q (examples: 500MB, 1.5GB, 200)", s)
	}

	bytes := v * float64(mult)
	if bytes > math.MaxInt64 {
		return 0, fmt.Errorf("size %q is too large", s)
	}

	return int64(bytes), nil
}

// ParseWithin parses a look-back period: "7" (days), "3d", "2w", "24h", "90m", "1h30m".
func ParseWithin(s string) (time.Duration, error) {
	num, unit := splitNumber(s)

	var mult time.Duration

	switch unit {
	case "", "d", "д":
		mult = 24 * time.Hour
	case "w", "н":
		mult = 7 * 24 * time.Hour
	case "h", "ч":
		mult = time.Hour
	}

	if mult != 0 && num != "" {
		v, err := strconv.ParseFloat(num, 64)
		if err == nil && v > 0 {
			return time.Duration(v * float64(mult)), nil
		}
	}

	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid period %q (examples: 24h, 3d, 2w, 7)", s)
	}

	return d, nil
}

// splitNumber splits "1,5 GB" into ("1.5", "gb").
func splitNumber(s string) (num, unit string) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")

	i := strings.IndexFunc(s, func(r rune) bool {
		return (r < '0' || r > '9') && r != '.'
	})
	if i < 0 {
		return s, ""
	}

	return s[:i], s[i:]
}

// FormatAge renders a duration as "just now", "15m ago", "5h ago" or "3d ago".
func FormatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// FormatCount renders an integer with thousands separators: 1234567 -> "1,234,567".
func FormatCount(n int64) string {
	s := strconv.FormatInt(n, 10)

	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}

	var b strings.Builder

	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}

	if neg {
		return "-" + b.String()
	}

	return b.String()
}
