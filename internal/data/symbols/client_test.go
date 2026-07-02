package symbols

import (
	"context"
	"io"
	"net/http"
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

func TestLookupResolvesNasdaqNYSEAndSP500Tickers(t *testing.T) {
	t.Parallel()

	nasdaqBody := `Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares
AAPL|Apple Inc. Common Stock|Q|N|N|100|N|N
MSFT|Microsoft Corporation Common Stock|Q|N|N|100|N|N
AMZN|Amazon.com, Inc. Common Stock|Q|N|N|100|N|N
File Creation Time: 0701202612:01|||||||
`
	otherBody := `ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol
JPM|JPMorgan Chase & Co. Common Stock|N|JPM|N|100|N|JPM
XOM|Exxon Mobil Corporation Common Stock|N|XOM|N|100|N|XOM
File Creation Time: 0701202612:03|||||||
`
	client, cleanup := testSymbolClient(t, nasdaqBody, otherBody)
	defer cleanup()

	tests := map[string]string{
		"AAPL": "NASDAQ",
		"MSFT": "NASDAQ",
		"AMZN": "NASDAQ",
		"JPM":  "New York Stock Exchange",
		"XOM":  "New York Stock Exchange",
	}
	for ticker, wantExchange := range tests {
		listing, err := client.Lookup(context.Background(), ticker)
		if err != nil {
			t.Fatalf("Lookup(%s) returned error: %v", ticker, err)
		}
		if listing.Symbol != ticker || listing.Exchange != wantExchange {
			t.Fatalf("Lookup(%s) = %#v, want exchange %s", ticker, listing, wantExchange)
		}
	}
}

func TestLookupUsesCuratedFallbackWhenLiveDirectoriesMiss(t *testing.T) {
	t.Parallel()

	client, cleanup := testSymbolClient(t, emptyNasdaqDirectory(), emptyOtherDirectory())
	defer cleanup()

	for _, ticker := range []string{"TSM", "ASML", "BABA", "NVO"} {
		listing, err := client.Lookup(context.Background(), ticker)
		if err != nil {
			t.Fatalf("Lookup(%s) returned error: %v", ticker, err)
		}
		if listing.Symbol != ticker || listing.SecurityName == "" {
			t.Fatalf("Lookup(%s) returned incomplete listing: %#v", ticker, listing)
		}
	}
}

func TestLookupUnknownTickerReturnsUniverseError(t *testing.T) {
	t.Parallel()

	client, cleanup := testSymbolClient(t, emptyNasdaqDirectory(), emptyOtherDirectory())
	defer cleanup()

	_, err := client.Lookup(context.Background(), "ZZZZ")
	if err == nil {
		t.Fatal("Lookup accepted unknown ticker")
	}
	if !strings.Contains(err.Error(), "Ticker not found in the current Stockbridge symbol universe") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTickerVariantsAvoidDuplicates(t *testing.T) {
	t.Parallel()

	variants := TickerVariants("AAPL")
	if len(variants) != 1 || variants[0] != "AAPL" {
		t.Fatalf("TickerVariants(AAPL) = %#v, want only AAPL", variants)
	}
	variants = TickerVariants("BRK.B")
	if len(variants) != 2 || variants[0] != "BRK.B" || variants[1] != "BRK-B" {
		t.Fatalf("TickerVariants(BRK.B) = %#v", variants)
	}
}

func testSymbolClient(t *testing.T, nasdaqBody, otherBody string) (*Client, func()) {
	t.Helper()
	httpClient := &http.Client{Transport: symbolRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://test.stockbridge/nasdaqlisted.txt":
			return symbolTextResponse(http.StatusOK, nasdaqBody), nil
		case "https://test.stockbridge/otherlisted.txt":
			return symbolTextResponse(http.StatusOK, otherBody), nil
		default:
			return symbolTextResponse(http.StatusNotFound, ""), nil
		}
	})}
	client := NewClient(httpClient)
	client.nasdaqListedURL = "https://test.stockbridge/nasdaqlisted.txt"
	client.otherListedURL = "https://test.stockbridge/otherlisted.txt"
	return client, func() {}
}

type symbolRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn symbolRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func symbolTextResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func emptyNasdaqDirectory() string {
	return "Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares\nFile Creation Time: 0701202612:01|||||||\n"
}

func emptyOtherDirectory() string {
	return "ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol\nFile Creation Time: 0701202612:03|||||||\n"
}
