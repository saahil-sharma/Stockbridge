package symbols

import (
	"strings"
	"testing"
	"time"
)

func TestParseOtherListed(t *testing.T) {
	t.Parallel()

	input := strings.NewReader(`ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol
IBM|International Business Machines Corporation Common Stock|N|IBM|N|100|N|IBM
ABC|Example Test Issue|N|ABC|N|100|Y|ABC
File Creation Time: 0701202612:03|||||||
`)

	listings, fileCreatedAt, err := ParseOtherListed(input, "source", time.Unix(0, 0))
	if err != nil {
		t.Fatalf("ParseOtherListed returned error: %v", err)
	}
	if len(listings) != 2 {
		t.Fatalf("len(listings) = %d, want 2", len(listings))
	}
	if listings[0].Symbol != "IBM" || listings[0].Exchange != "New York Stock Exchange" || listings[0].Market != "" || listings[0].TestIssue {
		t.Fatalf("unexpected listing: %#v", listings[0])
	}
	if fileCreatedAt != "File Creation Time: 0701202612:03" {
		t.Fatalf("fileCreatedAt = %q", fileCreatedAt)
	}
}

func TestParseNasdaqListed(t *testing.T) {
	t.Parallel()

	input := strings.NewReader(`Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares
AMZN|Amazon.com, Inc. Common Stock|Q|N|N|100|N|N
ZZZ|Example Test Issue|S|Y|N|100|N|N
File Creation Time: 0701202612:01|||||||
`)

	listings, fileCreatedAt, err := ParseNasdaqListed(input, "source", time.Unix(0, 0))
	if err != nil {
		t.Fatalf("ParseNasdaqListed returned error: %v", err)
	}
	if len(listings) != 2 {
		t.Fatalf("len(listings) = %d, want 2", len(listings))
	}
	if listings[0].Symbol != "AMZN" || listings[0].Exchange != "NASDAQ" || listings[0].Market != "Nasdaq Global Select Market" || listings[0].TestIssue {
		t.Fatalf("unexpected listing: %#v", listings[0])
	}
	if fileCreatedAt != "File Creation Time: 0701202612:01" {
		t.Fatalf("fileCreatedAt = %q", fileCreatedAt)
	}
}

func TestCuratedFallbackListingResolvesMajorForeignIssuers(t *testing.T) {
	t.Parallel()

	tests := []string{"TSM", "ASML", "BABA", "NVO", "TCEHY", "SHEL", "RIO"}
	for _, ticker := range tests {
		ticker := ticker
		t.Run(ticker, func(t *testing.T) {
			t.Parallel()
			listing, found := CuratedFallbackListing(ticker)
			if !found {
				t.Fatalf("CuratedFallbackListing(%q) was not found", ticker)
			}
			if listing.Symbol != ticker || listing.SecurityName == "" || listing.Exchange == "" {
				t.Fatalf("unexpected listing for %s: %#v", ticker, listing)
			}
		})
	}
}

func TestEquivalentTickersHandlesClassPunctuation(t *testing.T) {
	t.Parallel()

	if !EquivalentTickers("BRK.B", "BRK-B") {
		t.Fatal("EquivalentTickers did not match BRK.B and BRK-B")
	}
	if !EquivalentTickers("BF-B", "BF.B") {
		t.Fatal("EquivalentTickers did not match BF-B and BF.B")
	}
	if EquivalentTickers("AAPL", "MSFT") {
		t.Fatal("EquivalentTickers matched unrelated tickers")
	}
}
