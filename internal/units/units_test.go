package units

import (
	"testing"
	"time"
)

func TestParseSize(t *testing.T) {
	cases := map[string]int64{
		"0":      0,
		"200":    200 * MB,
		"500MB":  500 * MB,
		"500 mb": 500 * MB,
		"1.5GB":  GB + GB/2,
		"1,5 гб": GB + GB/2,
		"2g":     2 * GB,
		"10k":    10 * KB,
		"1TB":    TB,
		"123b":   123,
		"64 MiB": 64 * MB,
	}

	for in, want := range cases {
		got, err := ParseSize(in)
		if err != nil {
			t.Errorf("ParseSize(%q): unexpected error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseSize(%q) = %d, want %d", in, got, want)
		}
	}

	for _, in := range []string{"", "abc", "10xb", "-5MB", "GB"} {
		if _, err := ParseSize(in); err == nil {
			t.Errorf("ParseSize(%q): expected error", in)
		}
	}
}

func TestParseWithin(t *testing.T) {
	day := 24 * time.Hour
	cases := map[string]time.Duration{
		"7":     7 * day,
		"3d":    3 * day,
		"2w":    14 * day,
		"24h":   24 * time.Hour,
		"90m":   90 * time.Minute,
		"1h30m": 90 * time.Minute,
		"0.5d":  12 * time.Hour,
	}

	for in, want := range cases {
		got, err := ParseWithin(in)
		if err != nil {
			t.Errorf("ParseWithin(%q): unexpected error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseWithin(%q) = %v, want %v", in, got, want)
		}
	}

	for _, in := range []string{"", "0", "-1d", "soon"} {
		if _, err := ParseWithin(in); err == nil {
			t.Errorf("ParseWithin(%q): expected error", in)
		}
	}
}

func TestFormatSize(t *testing.T) {
	cases := map[int64]string{
		0:         "0 B",
		1023:      "1023 B",
		1536:      "1.50 KB",
		5 * MB:    "5.00 MB",
		GB + GB/4: "1.25 GB",
		3 * TB:    "3.00 TB",
	}

	for in, want := range cases {
		if got := FormatSize(in); got != want {
			t.Errorf("FormatSize(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatCount(t *testing.T) {
	cases := map[int64]string{
		0:       "0",
		999:     "999",
		1000:    "1,000",
		1234567: "1,234,567",
		-1234:   "-1,234",
	}

	for in, want := range cases {
		if got := FormatCount(in); got != want {
			t.Errorf("FormatCount(%d) = %q, want %q", in, got, want)
		}
	}
}
