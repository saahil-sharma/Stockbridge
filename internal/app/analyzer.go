package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"stockbridge/internal/analysis"
	"stockbridge/internal/data/fundamentals"
	"stockbridge/internal/data/market"
	"stockbridge/internal/data/sec"
	"stockbridge/internal/data/symbols"
)

const marketSummaryTimeout = 8 * time.Second

var (
	ErrTickerNotFound  = errors.New("ticker not found")
	ErrDataUnavailable = errors.New("financial data unavailable")
)

type Analyzer struct {
	symbolClient       *symbols.Client
	secClient          *sec.Client
	marketClient       *market.Client
	fundamentalsClient *fundamentals.Client
}

func NewAnalyzer(httpClient *http.Client) *Analyzer {
	return &Analyzer{
		symbolClient:       symbols.NewClient(httpClient),
		secClient:          sec.NewClient(httpClient, os.Getenv("STOCKBRIDGE_SEC_USER_AGENT")),
		marketClient:       market.NewClient(httpClient),
		fundamentalsClient: fundamentals.NewClient(httpClient, os.Getenv("STOCKBRIDGE_FMP_API_KEY")),
	}
}

func (a *Analyzer) Analyze(ctx context.Context, rawTicker string) (analysis.Summary, error) {
	ticker, err := symbols.NormalizeTicker(rawTicker)
	if err != nil {
		return analysis.Summary{}, err
	}

	listing, listingErr := a.symbolClient.Lookup(ctx, ticker)
	if listingErr != nil {
		company, companyErr := a.secClient.LookupCompany(ctx, ticker)
		if companyErr != nil {
			if ctx.Err() != nil {
				return analysis.Summary{}, ctx.Err()
			}
			if errors.Is(listingErr, symbols.ErrTickerNotFound) && errors.Is(companyErr, sec.ErrCompanyNotFound) {
				return analysis.Summary{}, fmt.Errorf("%w: Ticker not found in the current Stockbridge symbol universe.", ErrTickerNotFound)
			}
			return analysis.Summary{}, fmt.Errorf("%w: symbol lookup failed: %v; SEC lookup failed: %v", ErrDataUnavailable, listingErr, companyErr)
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
			if summary, ok, fallbackErr := a.buildMarketSummary(ctx, listing, ticker); ok {
				return summary, nil
			} else if fallbackErr != nil {
				summary := analysis.BuildAvailabilitySummary(listing, ticker, &company, publicFallbackReason(factsErr, fallbackErr))
				return summary, nil
			}
			summary := analysis.BuildAvailabilitySummary(listing, ticker, &company, publicAvailabilityReason(factsErr))
			return summary, nil
		}
		summary := analysis.BuildSummary(listing, company, facts)
		if len(summary.Metrics) == 0 {
			if fallback, ok, _ := a.buildMarketSummary(ctx, listing, ticker); ok && len(fallback.Metrics) > 0 {
				return fallback, nil
			}
			summary.Notes = append(summary.Notes, fmt.Sprintf("Stockbridge recognizes %s as %s, but standardized SEC fundamentals are not available in the current local dataset. Try another ticker or update the fundamentals data source.", ticker, summary.CompanyName))
		}
		return a.addMarketNotes(ctx, ticker, facts, summary), nil
	}

	company, err := a.secClient.LookupCompany(ctx, ticker)
	if err != nil {
		if summary, ok, fallbackErr := a.buildMarketSummary(ctx, listing, ticker); ok {
			return summary, nil
		} else if fallbackErr != nil {
			summary := analysis.BuildAvailabilitySummary(listing, ticker, nil, publicFallbackReason(err, fallbackErr))
			return summary, nil
		}
		summary := analysis.BuildAvailabilitySummary(listing, ticker, nil, publicAvailabilityReason(err))
		return summary, nil
	}

	facts, err := a.secClient.CompanyFacts(ctx, company.CIK)
	if err != nil {
		if summary, ok, fallbackErr := a.buildMarketSummary(ctx, listing, ticker); ok {
			return summary, nil
		} else if fallbackErr != nil {
			summary := analysis.BuildAvailabilitySummary(listing, ticker, &company, publicFallbackReason(err, fallbackErr))
			return summary, nil
		}
		summary := analysis.BuildAvailabilitySummary(listing, ticker, &company, publicAvailabilityReason(err))
		return summary, nil
	}

	summary := analysis.BuildSummary(listing, company, facts)
	if len(summary.Metrics) == 0 {
		if fallback, ok, _ := a.buildMarketSummary(ctx, listing, ticker); ok && len(fallback.Metrics) > 0 {
			return fallback, nil
		}
		summary.Notes = append(summary.Notes, fmt.Sprintf("Stockbridge recognizes %s as %s, but standardized SEC fundamentals are not available in the current local dataset. Try another ticker or update the fundamentals data source.", ticker, summary.CompanyName))
	}
	return a.addMarketNotes(ctx, ticker, facts, summary), nil
}

func (a *Analyzer) buildMarketSummary(ctx context.Context, listing symbols.Listing, ticker string) (analysis.Summary, bool, error) {
	if a.fundamentalsClient == nil || !a.fundamentalsClient.Configured() {
		return analysis.Summary{}, false, nil
	}

	marketCtx, cancel := context.WithTimeout(ctx, marketSummaryTimeout)
	defer cancel()

	snapshot, err := a.fundamentalsClient.Snapshot(marketCtx, ticker)
	if err != nil {
		return analysis.Summary{}, false, err
	}

	latestPrice, err := a.marketClient.LatestClose(marketCtx, ticker)
	var price *market.LatestPrice
	if err == nil {
		price = &latestPrice
	}

	summary := analysis.BuildMarketSummary(listing, snapshot.Company, snapshot, price)
	if len(summary.Metrics) == 0 {
		return analysis.Summary{}, false, nil
	}
	if err != nil {
		summary.Notes = append(summary.Notes, "P/E ratio is temporarily unavailable because current price data could not be fetched.")
	}
	return summary, true, nil
}

func (a *Analyzer) addMarketNotes(ctx context.Context, ticker string, facts sec.CompanyFacts, summary analysis.Summary) analysis.Summary {
	latestPrice, err := a.marketClient.LatestClose(ctx, ticker)
	if err != nil {
		summary.Notes = append(summary.Notes, "P/E ratio is temporarily unavailable because current price data could not be fetched.")
	} else if !analysis.AddPERatio(&summary, facts, latestPrice) {
		summary.Notes = append(summary.Notes, "P/E ratio unavailable because positive annual diluted EPS was not found in SEC company-facts data.")
	}

	return summary
}

func publicAvailabilityReason(err error) string {
	if errors.Is(err, sec.ErrCompanyNotFound) {
		return "No matching SEC company-facts record was available."
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "A fundamentals provider timed out before returning data."
	}
	return "A fundamentals provider is temporarily unavailable or rate-limited."
}

func publicFallbackReason(primaryErr, fallbackErr error) string {
	fallbackReason := "The fallback fundamentals provider is also temporarily unavailable or rate-limited."
	if errors.Is(fallbackErr, context.DeadlineExceeded) || errors.Is(fallbackErr, context.Canceled) {
		fallbackReason = "The fallback fundamentals provider also timed out before returning data."
	}
	return publicAvailabilityReason(primaryErr) + " " + fallbackReason
}

func (a *Analyzer) PriceChart(ctx context.Context, rawTicker string) (market.Bundle, error) {
	ticker, err := symbols.NormalizeTicker(rawTicker)
	if err != nil {
		return market.Bundle{}, err
	}
	return a.marketClient.FetchBundle(ctx, ticker)
}
