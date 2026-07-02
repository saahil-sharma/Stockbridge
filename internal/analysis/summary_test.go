package analysis

import (
	"testing"
	"time"

	"stockbridge/internal/data/market"
	"stockbridge/internal/data/sec"
	"stockbridge/internal/data/symbols"
)

func TestBuildSummaryExtractsLatestMetric(t *testing.T) {
	t.Parallel()

	oldValue := 1_000.0
	newValue := 2_000.0
	summary := BuildSummary(
		symbols.Listing{Symbol: "IBM", SecurityName: "IBM Common Stock", Exchange: "New York Stock Exchange", SourceURL: "symbols", RetrievedAt: time.Unix(0, 0)},
		sec.Company{CIK: 51143, Ticker: "IBM", Title: "IBM", SourceURL: "companies", RetrievedAt: time.Unix(0, 0)},
		sec.CompanyFacts{
			EntityName: "International Business Machines Corp",
			SourceURL:  "facts",
			Facts: map[string]sec.TaxonomyFacts{
				"us-gaap": {
					"Revenues": {
						Units: map[string][]sec.FactUnit{
							"USD": {
								{Val: &oldValue, End: "2024-12-31", Form: "10-K", Filed: "2025-02-01"},
								{Val: &newValue, End: "2025-12-31", Form: "10-K", Filed: "2026-02-01"},
							},
						},
					},
				},
			},
		},
	)

	if len(summary.Metrics) != 1 {
		t.Fatalf("len(summary.Metrics) = %d, want 1", len(summary.Metrics))
	}
	if summary.Metrics[0].Value != newValue {
		t.Fatalf("metric value = %v, want %v", summary.Metrics[0].Value, newValue)
	}
}

func TestAddPERatio(t *testing.T) {
	t.Parallel()

	summary := Summary{
		Metrics: []Metric{},
	}
	annualEPS := 5.0
	quarterlyEPS := 1.0

	ok := AddPERatio(
		&summary,
		sec.CompanyFacts{
			Facts: map[string]sec.TaxonomyFacts{
				"us-gaap": {
					"EarningsPerShareDiluted": {
						Units: map[string][]sec.FactUnit{
							"USD/shares": {
								{Val: &quarterlyEPS, End: "2026-03-31", Form: "10-Q", Filed: "2026-04-30"},
								{Val: &annualEPS, End: "2025-12-31", Form: "10-K", Filed: "2026-02-15"},
							},
						},
					},
				},
			},
		},
		market.LatestPrice{
			Price:       125,
			Time:        time.Date(2026, 7, 1, 15, 30, 0, 0, time.UTC),
			SourceName:  "price source",
			SourceURL:   "https://example.com/price",
			RetrievedAt: time.Date(2026, 7, 1, 15, 31, 0, 0, time.UTC),
		},
	)
	if !ok {
		t.Fatal("AddPERatio returned false")
	}
	if len(summary.Metrics) != 1 {
		t.Fatalf("len(summary.Metrics) = %d, want 1", len(summary.Metrics))
	}
	if summary.Metrics[0].Name != "P/E ratio" || summary.Metrics[0].Value != 25 {
		t.Fatalf("unexpected P/E metric: %#v", summary.Metrics[0])
	}
	if len(summary.Sources) != 1 {
		t.Fatalf("len(summary.Sources) = %d, want 1", len(summary.Sources))
	}
}
