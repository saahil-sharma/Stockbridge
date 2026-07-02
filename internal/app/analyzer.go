package app

import (
	"context"
	"fmt"
	"net/http"

	"stockbridge/internal/analysis"
	"stockbridge/internal/data/market"
	"stockbridge/internal/data/sec"
	"stockbridge/internal/data/symbols"
)

type Analyzer struct {
	symbolClient *symbols.Client
	secClient    *sec.Client
	marketClient *market.Client
}

func NewAnalyzer(httpClient *http.Client) *Analyzer {
	return &Analyzer{
		symbolClient: symbols.NewClient(httpClient),
		secClient:    sec.NewClient(httpClient),
		marketClient: market.NewClient(httpClient),
	}
}

func (a *Analyzer) Analyze(ctx context.Context, rawTicker string) (analysis.Summary, error) {
	ticker, err := symbols.NormalizeTicker(rawTicker)
	if err != nil {
		return analysis.Summary{}, err
	}

	listing, err := a.symbolClient.Lookup(ctx, ticker)
	if err != nil {
		return analysis.Summary{}, err
	}

	company, err := a.secClient.LookupCompany(ctx, ticker)
	if err != nil {
		return analysis.Summary{}, err
	}

	facts, err := a.secClient.CompanyFacts(ctx, company.CIK)
	if err != nil {
		return analysis.Summary{}, err
	}

	summary := analysis.BuildSummary(listing, company, facts)
	latestPrice, err := a.marketClient.LatestClose(ctx, ticker)
	if err != nil {
		summary.Notes = append(summary.Notes, fmt.Sprintf("P/E ratio unavailable because latest price data could not be fetched: %v", err))
	} else if !analysis.AddPERatio(&summary, facts, latestPrice) {
		summary.Notes = append(summary.Notes, "P/E ratio unavailable because positive annual diluted EPS was not found in SEC company-facts data.")
	}

	return summary, nil
}

func (a *Analyzer) PriceChart(ctx context.Context, rawTicker string) (market.Bundle, error) {
	ticker, err := symbols.NormalizeTicker(rawTicker)
	if err != nil {
		return market.Bundle{}, err
	}
	return a.marketClient.FetchBundle(ctx, ticker)
}
