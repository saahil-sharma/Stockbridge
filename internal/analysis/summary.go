package analysis

import (
	"sort"
	"strings"

	"stockbridge/internal/data/sec"
	"stockbridge/internal/data/symbols"
)

type Summary struct {
	CompanyName string
	Ticker      string
	CIK         int
	Listing     symbols.Listing
	Metrics     []Metric
	Sources     []Source
	Notes       []string
}

type Metric struct {
	Name    string
	Value   float64
	Unit    string
	Period  string
	Form    string
	Filed   string
	Concept string
}

type Source struct {
	Name        string
	URL         string
	RetrievedAt string
}

func BuildSummary(listing symbols.Listing, company sec.Company, facts sec.CompanyFacts) Summary {
	summary := Summary{
		CompanyName: firstNonEmpty(facts.EntityName, company.Title, listing.SecurityName),
		Ticker:      company.Ticker,
		CIK:         company.CIK,
		Listing:     listing,
		Sources: []Source{
			{Name: "Nasdaq Trader symbol directory", URL: listing.SourceURL, RetrievedAt: listing.RetrievedAt.Format("2006-01-02 15:04:05 MST")},
			{Name: "SEC company tickers", URL: company.SourceURL, RetrievedAt: company.RetrievedAt.Format("2006-01-02 15:04:05 MST")},
			{Name: "SEC company facts", URL: facts.SourceURL, RetrievedAt: facts.RetrievedAt.Format("2006-01-02 15:04:05 MST")},
		},
		Notes: []string{
			"Report uses public listing and SEC XBRL company-facts data only.",
			"Historical chart code is currently sidelined for the future app UI; valuation ratios and recent news are planned provider integrations.",
			"This output is informational and is not personalized financial advice.",
		},
	}

	for _, spec := range metricSpecs {
		if metric, ok := latestMetric(facts, spec); ok {
			summary.Metrics = append(summary.Metrics, metric)
		}
	}

	return summary
}

type metricSpec struct {
	Name     string
	Unit     string
	Concepts []string
}

var metricSpecs = []metricSpec{
	{Name: "Revenue", Unit: "USD", Concepts: []string{"RevenueFromContractWithCustomerExcludingAssessedTax", "Revenues"}},
	{Name: "Net income", Unit: "USD", Concepts: []string{"NetIncomeLoss", "ProfitLoss"}},
	{Name: "Assets", Unit: "USD", Concepts: []string{"Assets"}},
	{Name: "Liabilities", Unit: "USD", Concepts: []string{"Liabilities"}},
	{Name: "Stockholders' equity", Unit: "USD", Concepts: []string{"StockholdersEquity", "StockholdersEquityIncludingPortionAttributableToNoncontrollingInterest"}},
	{Name: "Cash and equivalents", Unit: "USD", Concepts: []string{"CashAndCashEquivalentsAtCarryingValue", "CashCashEquivalentsRestrictedCashAndRestrictedCashEquivalents"}},
	{Name: "Operating cash flow", Unit: "USD", Concepts: []string{"NetCashProvidedByUsedInOperatingActivities"}},
	{Name: "Capital expenditures", Unit: "USD", Concepts: []string{"PaymentsToAcquirePropertyPlantAndEquipment"}},
	{Name: "Basic EPS", Unit: "USD/shares", Concepts: []string{"EarningsPerShareBasic"}},
	{Name: "Diluted EPS", Unit: "USD/shares", Concepts: []string{"EarningsPerShareDiluted"}},
}

func latestMetric(facts sec.CompanyFacts, spec metricSpec) (Metric, bool) {
	usgaap := facts.Facts["us-gaap"]
	if usgaap == nil {
		return Metric{}, false
	}

	for _, conceptName := range spec.Concepts {
		concept, ok := usgaap[conceptName]
		if !ok {
			continue
		}
		units := concept.Units[spec.Unit]
		if len(units) == 0 {
			continue
		}

		candidates := make([]sec.FactUnit, 0, len(units))
		for _, unit := range units {
			if unit.Val == nil || !isUsefulForm(unit.Form) {
				continue
			}
			candidates = append(candidates, unit)
		}
		if len(candidates) == 0 {
			continue
		}

		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].Filed == candidates[j].Filed {
				return candidates[i].End > candidates[j].End
			}
			return candidates[i].Filed > candidates[j].Filed
		})

		best := candidates[0]
		return Metric{
			Name:    spec.Name,
			Value:   *best.Val,
			Unit:    spec.Unit,
			Period:  best.End,
			Form:    best.Form,
			Filed:   best.Filed,
			Concept: conceptName,
		}, true
	}

	return Metric{}, false
}

func isUsefulForm(form string) bool {
	switch strings.ToUpper(form) {
	case "10-K", "10-Q", "20-F", "40-F":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "Unknown company"
}
