package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"stockbridge/internal/analysis"
	"stockbridge/internal/data/fundamentals"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestAnalyzeForeignIssuerUsesMarketFundamentalsWhenSECDataMissing(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.String(), "nasdaqlisted.txt"):
			return textResponse(http.StatusOK, emptyNasdaqDirectory()), nil
		case strings.Contains(req.URL.String(), "otherlisted.txt"):
			return textResponse(http.StatusOK, emptyOtherDirectory()), nil
		case strings.Contains(req.URL.String(), "company_tickers.json"):
			return textResponse(http.StatusOK, `{}`), nil
		case strings.Contains(req.URL.String(), "/profile/TSM"):
			return textResponse(http.StatusOK, `[{"symbol":"TSM","companyName":"Taiwan Semiconductor Manufacturing Company Limited","exchange":"NYSE","sector":"Technology","industry":"Semiconductors","description":"Leading semiconductor foundry","currency":"USD","mktCap":1000000000}]`), nil
		case strings.Contains(req.URL.String(), "/income-statement/TSM"):
			return textResponse(http.StatusOK, `[{"date":"2025-12-31","revenue":100000000,"grossProfit":60000000,"operatingIncome":30000000,"netIncome":20000000,"eps":1.50,"epsdiluted":1.40},{"date":"2024-12-31","revenue":80000000,"grossProfit":48000000,"operatingIncome":24000000,"netIncome":16000000,"eps":1.20,"epsdiluted":1.10}]`), nil
		case strings.Contains(req.URL.String(), "/balance-sheet-statement/TSM"):
			return textResponse(http.StatusOK, `[{"date":"2025-12-31","cashAndCashEquivalents":40000000,"totalAssets":200000000,"totalLiabilities":90000000,"totalStockholdersEquity":110000000,"totalDebt":25000000}]`), nil
		case strings.Contains(req.URL.String(), "/cash-flow-statement/TSM"):
			return textResponse(http.StatusOK, `[{"date":"2025-12-31","operatingCashFlow":35000000,"capitalExpenditure":-10000000,"freeCashFlow":25000000}]`), nil
		case strings.Contains(req.URL.String(), "/chart/TSM"):
			return textResponse(http.StatusOK, `{"chart":{"result":[{"timestamp":[1714478400],"indicators":{"quote":[{"close":[125.5]}]}}],"error":null}}`), nil
		default:
			return textResponse(http.StatusNotFound, `{}`), nil
		}
	})}
	analyzer := NewAnalyzer(httpClient)
	analyzer.fundamentalsClient = fundamentals.NewClient(httpClient, "test-key")

	summary, err := analyzer.Analyze(context.Background(), "TSM")
	if err != nil {
		t.Fatalf("Analyze(TSM) returned error: %v", err)
	}
	if summary.Ticker != "TSM" || !strings.Contains(summary.CompanyName, "Taiwan Semiconductor") {
		t.Fatalf("unexpected summary identity: %#v", summary)
	}
	if len(summary.Metrics) == 0 {
		t.Fatalf("expected market fundamentals, got none: %#v", summary)
	}
	if !metricNamesContain(summary.Metrics, "Revenue", "Net income", "Operating cash flow", "P/E ratio") {
		t.Fatalf("missing expected fallback metrics: %#v", summary.Metrics)
	}
	if !notesContain(summary.Notes, "market-data fundamentals") {
		t.Fatalf("missing market-data note: %#v", summary.Notes)
	}
}

func TestAnalyzeForeignIssuerSkipsMarketFundamentalsWithoutAPIKey(t *testing.T) {
	t.Parallel()

	var fmpCalls atomic.Int32
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.String(), "nasdaqlisted.txt"):
			return textResponse(http.StatusOK, emptyNasdaqDirectory()), nil
		case strings.Contains(req.URL.String(), "otherlisted.txt"):
			return textResponse(http.StatusOK, emptyOtherDirectory()), nil
		case strings.Contains(req.URL.String(), "company_tickers.json"):
			return textResponse(http.StatusOK, `{}`), nil
		case strings.Contains(req.URL.Host, "financialmodelingprep.com"):
			fmpCalls.Add(1)
			return textResponse(http.StatusOK, `[]`), nil
		default:
			return textResponse(http.StatusNotFound, `{}`), nil
		}
	})}
	analyzer := NewAnalyzer(httpClient)
	analyzer.fundamentalsClient = fundamentals.NewClient(httpClient, "")

	summary, err := analyzer.Analyze(context.Background(), "TSM")
	if err != nil {
		t.Fatalf("Analyze(TSM) returned error: %v", err)
	}
	if fmpCalls.Load() != 0 {
		t.Fatalf("expected no FMP calls without an API key, got %d", fmpCalls.Load())
	}
	if len(summary.Metrics) != 0 {
		t.Fatalf("expected availability-only summary without configured fallback, got metrics: %#v", summary.Metrics)
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
	if !errors.Is(err, ErrTickerNotFound) {
		t.Fatalf("expected ErrTickerNotFound, got %v", err)
	}
}

func TestAnalyzeProviderFailureIsNotReportedAsUnknownTicker(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return textResponse(http.StatusTooManyRequests, `{}`), nil
	})})

	_, err := analyzer.Analyze(context.Background(), "AMZN")
	if err == nil {
		t.Fatal("Analyze accepted unavailable provider responses")
	}
	if !errors.Is(err, ErrDataUnavailable) {
		t.Fatalf("expected ErrDataUnavailable, got %v", err)
	}
	if strings.Contains(err.Error(), "Ticker not found in the current Stockbridge symbol universe") {
		t.Fatalf("provider failure was misreported as an unknown ticker: %v", err)
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

func metricNamesContain(metrics []analysis.Metric, names ...string) bool {
	want := make(map[string]struct{}, len(names))
	for _, name := range names {
		want[name] = struct{}{}
	}
	for _, metric := range metrics {
		delete(want, metric.Name)
	}
	return len(want) == 0
}

func emptyNasdaqDirectory() string {
	return "Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares\nFile Creation Time: 0701202612:01|||||||\n"
}

func emptyOtherDirectory() string {
	return "ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol\nFile Creation Time: 0701202612:03|||||||\n"
}
