package sec

import (
	"strings"
	"testing"
	"time"
)

func TestParseCompanies(t *testing.T) {
	t.Parallel()

	input := strings.NewReader(`{
		"0": {"cik_str": 51143, "ticker": "IBM", "title": "International Business Machines Corp"}
	}`)

	companies, err := ParseCompanies(input, "source", time.Unix(0, 0))
	if err != nil {
		t.Fatalf("ParseCompanies returned error: %v", err)
	}
	if len(companies) != 1 {
		t.Fatalf("len(companies) = %d, want 1", len(companies))
	}
	if companies[0].CIK != 51143 || companies[0].Ticker != "IBM" {
		t.Fatalf("unexpected company: %#v", companies[0])
	}
}
