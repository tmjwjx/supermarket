package main

import (
	"strings"
	"testing"
)

func TestInitPassword(t *testing.T) {
	cases := map[string]string{
		"":            "required",
		"   ":         "required",
		"admin12":     "at least 8",
		"口令口令口令口令":    "",
		" admin123 ":  "",
		"S3cure-pass": "",
	}
	for raw, want := range cases {
		got, err := initPassword(raw)
		if want == "" {
			if err != nil || got != strings.TrimSpace(raw) {
				t.Fatalf("%q: got %q %v", raw, got, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%q: got %v want %s", raw, err, want)
		}
	}
}
