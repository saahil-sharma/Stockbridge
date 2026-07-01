package analysis

import (
	"testing"
	"time"

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
