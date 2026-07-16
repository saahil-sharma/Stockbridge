package web

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"stockbridge/internal/analysis"
	"stockbridge/internal/app"
	"stockbridge/internal/data/market"
	"stockbridge/internal/data/symbols"
)

type fakeAnalyzer struct {
	analyze func(context.Context, string) (analysis.Summary, error)
	chart   func(context.Context, string) (market.Bundle, error)
}

func (f fakeAnalyzer) Analyze(ctx context.Context, ticker string) (analysis.Summary, error) {
	if f.analyze == nil {
		return analysis.Summary{}, nil
	}
	return f.analyze(ctx, ticker)
}

func (f fakeAnalyzer) PriceChart(ctx context.Context, ticker string) (market.Bundle, error) {
	if f.chart == nil {
		return market.Bundle{}, errors.New("chart unavailable")
	}
	return f.chart(ctx, ticker)
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	server := testServer(fakeAnalyzer{})
	recorder := httptest.NewRecorder()
	server.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d", recorder.Code)
	}
	if recorder.Body.String() != "ok\n" {
		t.Fatalf("GET /health body = %q", recorder.Body.String())
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("GET /health missing security headers: %#v", recorder.Header())
	}
}

func TestPublicAnalysisErrorsAreSafe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantText   string
	}{
		{name: "invalid ticker", err: symbols.ErrInvalidTicker, wantStatus: http.StatusBadRequest, wantText: "Enter a valid ticker"},
		{name: "unknown ticker", err: app.ErrTickerNotFound, wantStatus: http.StatusNotFound, wantText: "Ticker not found"},
		{name: "provider unavailable", err: app.ErrDataUnavailable, wantStatus: http.StatusServiceUnavailable, wantText: "temporarily unavailable or rate-limited"},
		{name: "provider timeout", err: context.DeadlineExceeded, wantStatus: http.StatusGatewayTimeout, wantText: "took too long"},
		{name: "unexpected internal error", err: errors.New("database URL and top-secret-key"), wantStatus: http.StatusInternalServerError, wantText: "could not complete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := testServer(fakeAnalyzer{analyze: func(context.Context, string) (analysis.Summary, error) {
				return analysis.Summary{}, tt.err
			}})
			recorder := httptest.NewRecorder()
			server.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/?ticker=AMZN", nil))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			body := recorder.Body.String()
			if !strings.Contains(body, tt.wantText) {
				t.Fatalf("body did not contain %q", tt.wantText)
			}
			if strings.Contains(body, "top-secret-key") || strings.Contains(body, "database URL") {
				t.Fatalf("body exposed an internal error: %s", body)
			}
		})
	}
}

func TestChartFailureIsRenderedAsPublicNote(t *testing.T) {
	t.Parallel()

	const secret = "top-secret-key"
	server := testServer(fakeAnalyzer{
		analyze: func(context.Context, string) (analysis.Summary, error) {
			return analysis.Summary{
				CompanyName: "Amazon.com, Inc.",
				Ticker:      "AMZN",
				Listing: symbols.Listing{
					Symbol:       "AMZN",
					SecurityName: "Amazon.com, Inc. Common Stock",
					Exchange:     "NASDAQ",
				},
				Notes: []string{"Current report data is source-attributed."},
			}, nil
		},
		chart: func(context.Context, string) (market.Bundle, error) {
			return market.Bundle{}, errors.New("upstream URL contained " + secret)
		},
	})
	recorder := httptest.NewRecorder()
	server.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/?ticker=AMZN", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "price chart is temporarily unavailable or rate-limited") {
		t.Fatalf("body did not contain the public chart note")
	}
	if strings.Contains(body, secret) {
		t.Fatalf("body exposed chart error details")
	}
}

func TestBusyServerReturnsRetryableResponse(t *testing.T) {
	t.Parallel()

	server := testServer(fakeAnalyzer{})
	for i := 0; i < maxConcurrentAnalyses; i++ {
		server.analysisSlots <- struct{}{}
	}
	recorder := httptest.NewRecorder()
	server.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/?ticker=AMZN", nil))

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTooManyRequests)
	}
	if recorder.Header().Get("Retry-After") != "5" {
		t.Fatalf("Retry-After = %q", recorder.Header().Get("Retry-After"))
	}
}

func TestBuildChartViewSkipsIncompleteSeries(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	view := buildChartView(market.Bundle{
		Ticker: "AMZN",
		Series: map[market.Period]market.Series{
			market.PeriodOneDay: {Period: market.PeriodOneDay, Points: []market.Point{{Time: now, Close: 100}}},
		},
	})
	if len(view.Periods) != 0 {
		t.Fatalf("expected incomplete chart period to be skipped: %#v", view.Periods)
	}
}

func testServer(analyzer Analyzer) *Server {
	server := NewServer(analyzer)
	server.logger = log.New(io.Discard, "", 0)
	return server
}
