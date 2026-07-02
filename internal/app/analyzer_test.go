package app

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestAnalyzeRecognizedForeignTickerWithMissingSECDataReturnsPartialReport(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.String(), "nasdaqlisted.txt"):
			return textResponse(http.StatusOK, emptyNasdaqDirectory()), nil
		case strings.Contains(req.URL.String(), "otherlisted.txt"):
			return textResponse(http.StatusOK, emptyOtherDirectory()), nil
		case strings.Contains(req.URL.String(), "company_tickers.json"):
			return textResponse(http.StatusOK, `{}`), nil
		default:
			return textResponse(http.StatusNotFound, `{}`), nil
		}
	})})

	summary, err := analyzer.Analyze(context.Background(), "TSM")
	if err != nil {
		t.Fatalf("Analyze(TSM) returned error: %v", err)
	}
	if summary.Ticker != "TSM" || !strings.Contains(summary.CompanyName, "Taiwan Semiconductor") {
		t.Fatalf("unexpected summary identity: %#v", summary)
	}
	if len(summary.Metrics) != 0 {
		t.Fatalf("partial report unexpectedly had metrics: %#v", summary.Metrics)
	}
	if !notesContain(summary.Notes, "standardized SEC fundamentals are not available") {
		t.Fatalf("missing availability note: %#v", summary.Notes)
	}
}

func TestAnalyzeUnknownTickerReturnsUniverseError(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.String(), "nasdaqlisted.txt"):
			return textResponse(http.StatusOK, emptyNasdaqDirectory()), nil
		case strings.Contains(req.URL.String(), "otherlisted.txt"):
			return textResponse(http.StatusOK, emptyOtherDirectory()), nil
		case strings.Contains(req.URL.String(), "company_tickers.json"):
			return textResponse(http.StatusOK, `{}`), nil
		default:
			return textResponse(http.StatusNotFound, `{}`), nil
		}
	})})

	_, err := analyzer.Analyze(context.Background(), "ZZZZ")
	if err == nil {
		t.Fatal("Analyze accepted unknown ticker")
	}
	if !strings.Contains(err.Error(), "Ticker not found in the current Stockbridge symbol universe") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func textResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       ioNopCloser{strings.NewReader(body)},
	}
}

type ioNopCloser struct {
	*strings.Reader
}

func (c ioNopCloser) Close() error {
	return nil
}

func notesContain(notes []string, needle string) bool {
	for _, note := range notes {
		if strings.Contains(note, needle) {
			return true
		}
	}
	return false
}

func emptyNasdaqDirectory() string {
	return "Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares\nFile Creation Time: 0701202612:01|||||||\n"
}

func emptyOtherDirectory() string {
	return "ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol\nFile Creation Time: 0701202612:03|||||||\n"
}
