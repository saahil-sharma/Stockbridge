package symbols

import "testing"

func TestNormalizeTicker(t *testing.T) {
	t.Parallel()

	got, err := NormalizeTicker(" ibm ")
	if err != nil {
		t.Fatalf("NormalizeTicker returned error: %v", err)
	}
	if got != "IBM" {
		t.Fatalf("NormalizeTicker = %q, want IBM", got)
	}
}

func TestNormalizeTickerRejectsInvalid(t *testing.T) {
	t.Parallel()

	if _, err := NormalizeTicker("bad ticker"); err == nil {
		t.Fatal("NormalizeTicker accepted invalid ticker")
	}
}
