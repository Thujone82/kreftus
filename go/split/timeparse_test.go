package main

import (
	"math"
	"testing"
)

func TestParseDurationSpec(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"45s", 45},
		{"2m", 120},
		{"1m3.2s", 63.2},
		{"1h2m3.4s", 3723.4},
		{"1:03.2", 63.2},
		{"0:01:03.2", 63.2},
		{"63.2", 63.2},
		{" 1M 3.2S ", 63.2},
	}
	for _, tc := range cases {
		got, err := ParseDurationSpec(tc.in)
		if err != nil {
			t.Fatalf("ParseDurationSpec(%q) unexpected err: %v", tc.in, err)
		}
		if math.Abs(got-tc.want) > 1e-6 {
			t.Fatalf("ParseDurationSpec(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}

	bad := []string{"", "-5s", "abc", "1:2:3:4", "5s2m", "1m-3s"}
	for _, b := range bad {
		if _, err := ParseDurationSpec(b); err == nil {
			t.Fatalf("ParseDurationSpec(%q) expected error, got nil", b)
		}
	}
}

func TestParseRangeList(t *testing.T) {
	ranges, err := ParseRangeList("start..15s, 0:30-1m10s, 1m30s-end", 120)
	if err != nil {
		t.Fatalf("ParseRangeList err: %v", err)
	}
	if len(ranges) != 3 {
		t.Fatalf("len(ranges) = %d, want 3", len(ranges))
	}
	if ranges[0].Start != 0 || ranges[0].End != 15 {
		t.Fatalf("range[0] = %+v", ranges[0])
	}
	if ranges[1].Start != 30 || ranges[1].End != 70 {
		t.Fatalf("range[1] = %+v", ranges[1])
	}
	if ranges[2].Start != 90 || ranges[2].End != 120 {
		t.Fatalf("range[2] = %+v", ranges[2])
	}
}
