package ui

import "testing"

func TestTruncateRuneSafe(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 3, "he…"},
		{"İşler", 4, "İşl…"},
		{"görev", 5, "görev"},
		{"görev", 1, "…"},
		{"x", 0, ""},
	}
	for _, c := range cases {
		if got := truncate(c.in, c.max); got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestClassifyPriority(t *testing.T) {
	cases := map[string]priorityKind{
		"Kritik":  prioUrgent,
		"Urgent":  prioUrgent,
		"Yüksek":  prioHigh,
		"High":    prioHigh,
		"Normal":  prioNormal,
		"Orta":    prioNormal,
		"Düşük":   prioLow,
		"Low":     prioLow,
		"unknown": prioLow,
	}
	for name, want := range cases {
		if got := classifyPriority(name); got != want {
			t.Errorf("classifyPriority(%q) = %d, want %d", name, got, want)
		}
	}
}

func TestValidHours(t *testing.T) {
	valid := []string{"1", "1.5", "0.25", "12", "2,5"}
	invalid := []string{"", "abc", "1.5.5", "1h", "-2", "."}
	for _, s := range valid {
		if !validHours(s) {
			t.Errorf("validHours(%q) = false, want true", s)
		}
	}
	for _, s := range invalid {
		if validHours(s) {
			t.Errorf("validHours(%q) = true, want false", s)
		}
	}
}
