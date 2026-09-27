package employees

import "testing"

func TestParseDateValueISOString(t *testing.T) {
	got, err := ParseDateValue("2026-09-15")
	if err != nil {
		t.Fatalf("ParseDateValue: %v", err)
	}
	if got != "2026-09-15" {
		t.Fatalf("got %q", got)
	}
}

func TestParseDateValueDDMMYYYYString(t *testing.T) {
	got, err := ParseDateValue("15-09-2026")
	if err != nil {
		t.Fatalf("ParseDateValue: %v", err)
	}
	if got != "2026-09-15" {
		t.Fatalf("got %q, want 2026-09-15", got)
	}
}

func TestParseDateValueExcelSerial(t *testing.T) {
	// A known modern date: 15 Sep 2026 is Excel serial 46280. (The 30 Dec
	// 1899 epoch used here is the standard, universally-replicated
	// simplification: it's off by one day for serials before 1 Mar 1900,
	// reproducing Excel's 1900 leap-year bug, but exactly correct for
	// every real-world date since — which is all that matters here.)
	got2, err := ParseDateValue(float64(46280))
	if err != nil {
		t.Fatalf("ParseDateValue: %v", err)
	}
	if got2 != "2026-09-15" {
		t.Fatalf("got %q, want 2026-09-15", got2)
	}
}

func TestParseDateValueRejectsGarbage(t *testing.T) {
	cases := []any{"", nil, "not a date", "2026/09/15", float64(0), float64(-5), true}
	for _, c := range cases {
		if _, err := ParseDateValue(c); err == nil {
			t.Errorf("ParseDateValue(%#v) should have failed", c)
		}
	}
}
