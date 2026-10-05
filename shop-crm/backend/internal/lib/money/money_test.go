package money

import "testing"

func TestRubToKop(t *testing.T) {
	valid := map[string]int64{
		"0":       0,
		"1":       100,
		"12":      1200,
		"12.3":    1230,
		"12.34":   1234,
		"12,34":   1234,
		"0.05":    5,
		"0.5":     50,
		"1234.99": 123499,
	}
	for input, want := range valid {
		got, err := RubToKop(input)
		if err != nil {
			t.Fatalf("RubToKop(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("RubToKop(%q) = %d, want %d", input, got, want)
		}
	}

	invalid := []string{"", "abc", "1.2.3", "1.234", "-5", "5-", " 12", "12 ", "1 000", "12."}
	for _, input := range invalid {
		if got, err := RubToKop(input); err == nil {
			t.Fatalf("RubToKop(%q) should have failed, got %d", input, got)
		}
	}
}

func TestFormatKop(t *testing.T) {
	tests := map[int64]string{
		0:           "0 ₽",
		5:           "0,05 ₽",
		50:          "0,50 ₽",
		100:         "1 ₽",
		1234:        "12,34 ₽",
		100000:      "1\u202f000 ₽",
		123400:      "1\u202f234 ₽",
		100000000:   "1\u202f000\u202f000 ₽",
		123400000:   "1\u202f234\u202f000 ₽",
		12340000000: "123\u202f400\u202f000 ₽",
		34950:       "349,50 ₽",
		1:           "0,01 ₽",
		123456:      "1\u202f234,56 ₽",
		-50000:      "-500 ₽",
	}
	for kop, want := range tests {
		if got := FormatKop(kop); got != want {
			t.Fatalf("FormatKop(%d) = %q, want %q", kop, got, want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	// A ruble string survives the trip through kopecks and back, with the kopeck part
	// added explicitly where the input omitted it.
	tests := map[string]string{
		"0":       "0 ₽",
		"1":       "1 ₽",
		"19.9":    "19,90 ₽",
		"19.99":   "19,99 ₽",
		"1234.56": "1\u202f234,56 ₽",
	}
	for input, want := range tests {
		kop, err := RubToKop(input)
		if err != nil {
			t.Fatalf("RubToKop(%q): %v", input, err)
		}
		if got := FormatKop(kop); got != want {
			t.Fatalf("round trip of %q produced %q, want %q", input, got, want)
		}
	}
}