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
		company, companyErr := a.secClient.LookupCompany(ctx, ticker)
		if companyErr != nil {
			return analysis.Summary{}, fmt.Errorf("Ticker not found in the current Stockbridge symbol universe.")
		}
		listing = symbols.Listing{
			Symbol:       ticker,
			SecurityName: company.Title,
			Exchange:     "SEC reporting company",
			Market:       "SEC company_tickers.json",
			SourceURL:    company.SourceURL,
			RetrievedAt:  company.RetrievedAt,
		}

		facts, factsErr := a.secClient.CompanyFacts(ctx, company.CIK)
		if factsErr != nil {
			summary := analysis.BuildAvailabilitySummary(listing, ticker, &company, factsErr.Error())
			return summary, nil
		}
		summary := analysis.BuildSummary(listing, company, facts)
		if len(summary.Metrics) == 0 {
			summary.Notes = append(summary.Notes, fmt.Sprintf("Stockbridge recognizes %s as %s, but standardized SEC fundamentals are not available in the current local dataset. Try another ticker or update the fundamentals data source.", ticker, summary.CompanyName))
		}
		return a.addMarketNotes(ctx, ticker, facts, summary), nil
	}

	company, err := a.secClient.LookupCompany(ctx, ticker)
	if err != nil {
		summary := analysis.BuildAvailabilitySummary(listing, ticker, nil, err.Error())
		return summary, nil
	}

	facts, err := a.secClient.CompanyFacts(ctx, company.CIK)
	if err != nil {
		summary := analysis.BuildAvailabilitySummary(listing, ticker, &company, err.Error())
		return summary, nil
	}

	summary := analysis.BuildSummary(listing, company, facts)
	if len(summary.Metrics) == 0 {
		summary.Notes = append(summary.Notes, fmt.Sprintf("Stockbridge recognizes %s as %s, but standardized SEC fundamentals are not available in the current local dataset. Try another ticker or update the fundamentals data source.", ticker, summary.CompanyName))
	}
	return a.addMarketNotes(ctx, ticker, facts, summary), nil
}

func (a *Analyzer) addMarketNotes(ctx context.Context, ticker string, facts sec.CompanyFacts, summary analysis.Summary) analysis.Summary {
	latestPrice, err := a.marketClient.LatestClose(ctx, ticker)
	if err != nil {
		summary.Notes = append(summary.Notes, fmt.Sprintf("P/E ratio unavailable because latest price data could not be fetched: %v", err))
	} else if !analysis.AddPERatio(&summary, facts, latestPrice) {
		summary.Notes = append(summary.Notes, "P/E ratio unavailable because positive annual diluted EPS was not found in SEC company-facts data.")
	}

	return summary
}

func (a *Analyzer) PriceChart(ctx context.Context, rawTicker string) (market.Bundle, error) {
	ticker, err := symbols.NormalizeTicker(rawTicker)
	if err != nil {
		return market.Bundle{}, err
	}
	return a.marketClient.FetchBundle(ctx, ticker)
}
