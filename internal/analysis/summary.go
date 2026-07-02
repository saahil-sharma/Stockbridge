package analysis

import (
	"sort"
	"strings"

	"stockbridge/internal/data/fundamentals"
	"stockbridge/internal/data/market"
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
	summary := buildSummary(
		listing,
		firstNonEmpty(facts.EntityName, company.Title, listing.SecurityName),
		company.Ticker,
		company.CIK,
		[]Source{
			{Name: "Nasdaq Trader symbol directory", URL: listing.SourceURL, RetrievedAt: listing.RetrievedAt.Format("2006-01-02 15:04:05 MST")},
			{Name: "SEC company tickers", URL: company.SourceURL, RetrievedAt: company.RetrievedAt.Format("2006-01-02 15:04:05 MST")},
			{Name: "SEC company facts", URL: facts.SourceURL, RetrievedAt: facts.RetrievedAt.Format("2006-01-02 15:04:05 MST")},
		},
		[]string{
			"Report uses public listing and SEC XBRL company-facts data when available.",
			"P/E ratio is derived from the latest fetched close and SEC annual diluted EPS when both are available.",
			"Historical price charts are available in the web UI; additional valuation ratios and recent news are planned provider integrations.",
			"This output is informational and is not personalized financial advice.",
		},
		nil,
	)

	for _, spec := range metricSpecs {
		if metric, ok := latestMetric(facts, spec); ok {
			summary.Metrics = append(summary.Metrics, metric)
		}
	}

	return summary
}

func BuildMarketSummary(listing symbols.Listing, company fundamentals.Company, snapshot fundamentals.Snapshot, latestPrice *market.LatestPrice) Summary {
	sources := []Source{
		{Name: "Nasdaq Trader symbol directory", URL: listing.SourceURL, RetrievedAt: listing.RetrievedAt.Format("2006-01-02 15:04:05 MST")},
		{Name: "Financial Modeling Prep profile", URL: snapshot.Company.SourceURL, RetrievedAt: snapshot.Company.RetrievedAt.Format("2006-01-02 15:04:05 MST")},
	}
	if len(snapshot.Sources) > 1 {
		for _, source := range snapshot.Sources[1:] {
			sources = append(sources, Source{Name: source.Name, URL: source.URL, RetrievedAt: source.RetrievedAt.Format("2006-01-02 15:04:05 MST")})
		}
	}
	if latestPrice != nil {
		sources = append(sources, Source{Name: latestPrice.SourceName, URL: latestPrice.SourceURL, RetrievedAt: latestPrice.RetrievedAt.Format("2006-01-02 15:04:05 MST")})
	}

	summary := buildSummary(
		listing,
		firstNonEmpty(snapshot.Company.Name, company.Name, listing.SecurityName),
		snapshot.Company.Ticker,
		0,
		sources,
		append([]string{
			"Report uses public listing and market-data fundamentals when SEC company-facts data are unavailable.",
			"Revenue, cash flow, leverage, and valuation ratios are normalized from provider statements and the latest market close.",
		}, snapshot.Notes...),
		nil,
	)

	for _, metric := range snapshot.Metrics {
		summary.Metrics = append(summary.Metrics, convertMetric(metric))
	}
	appendMarketValuations(&summary, snapshot, latestPrice)
	return summary
}

func BuildAvailabilitySummary(listing symbols.Listing, ticker string, company *sec.Company, reason string) Summary {
	companyName := listing.SecurityName
	cik := 0
	sources := []Source{
		{Name: "Symbol universe", URL: listing.SourceURL, RetrievedAt: listing.RetrievedAt.Format("2006-01-02 15:04:05 MST")},
	}
	if company != nil {
		companyName = firstNonEmpty(company.Title, companyName)
		ticker = firstNonEmpty(company.Ticker, ticker)
		cik = company.CIK
		sources = append(sources, Source{Name: "SEC company tickers", URL: company.SourceURL, RetrievedAt: company.RetrievedAt.Format("2006-01-02 15:04:05 MST")})
	}
	ticker = strings.ToUpper(strings.TrimSpace(ticker))

	note := "Stockbridge recognizes " + ticker + " as " + companyName + ", but standardized SEC fundamentals are not available in the current local dataset."
	if reason != "" {
		note += " Details: " + reason
	}
	note += " Try another ticker or update the fundamentals data source."

	return Summary{
		CompanyName: companyName,
		Ticker:      ticker,
		CIK:         cik,
		Listing:     listing,
		Sources:     sources,
		Notes: []string{
			note,
			"Report uses public listing and standardized fundamentals data when available.",
			"Historical price charts are available when market price data is available for this ticker.",
			"This output is informational and is not personalized financial advice.",
		},
	}
}

func buildSummary(listing symbols.Listing, companyName, ticker string, cik int, sources []Source, notes []string, metrics []Metric) Summary {
	return Summary{
		CompanyName: companyName,
		Ticker:      ticker,
		CIK:         cik,
		Listing:     listing,
		Metrics:     metrics,
		Sources:     sources,
		Notes:       notes,
	}
}

func convertMetric(metric fundamentals.Metric) Metric {
	return Metric{
		Name:    metric.Name,
		Value:   metric.Value,
		Unit:    metric.Unit,
		Period:  metric.Period,
		Form:    metric.Source,
		Concept: metric.Concept,
	}
}

func appendMarketValuations(summary *Summary, snapshot fundamentals.Snapshot, latestPrice *market.LatestPrice) {
	if summary == nil {
		return
	}

	revenue, ok := metricValue(summary.Metrics, "Revenue")
	if ok && snapshot.Company.MarketCap > 0 && revenue > 0 {
		summary.Metrics = append(summary.Metrics, Metric{
			Name:    "P/S ratio",
			Value:   snapshot.Company.MarketCap / revenue,
			Unit:    "x",
			Period:  latestMetricPeriod(summary.Metrics),
			Form:    "market cap / revenue",
			Concept: "marketCap/revenue",
		})
	}

	equity, ok := metricValue(summary.Metrics, "Stockholders' equity")
	if ok && snapshot.Company.MarketCap > 0 && equity > 0 {
		summary.Metrics = append(summary.Metrics, Metric{
			Name:    "P/B ratio",
			Value:   snapshot.Company.MarketCap / equity,
			Unit:    "x",
			Period:  latestMetricPeriod(summary.Metrics),
			Form:    "market cap / equity",
			Concept: "marketCap/equity",
		})
	}

	debt, debtOK := metricValue(summary.Metrics, "Total debt")
	if debtOK && equity > 0 {
		summary.Metrics = append(summary.Metrics, Metric{
			Name:    "Debt/equity ratio",
			Value:   debt / equity,
			Unit:    "x",
			Period:  latestMetricPeriod(summary.Metrics),
			Form:    "balance sheet",
			Concept: "debt/equity",
		})
	}

	if snapshot.Company.MarketCap > 0 && debtOK {
		cash, cashOK := metricValue(summary.Metrics, "Cash and equivalents")
		ev := snapshot.Company.MarketCap + debt
		if cashOK {
			ev -= cash
		}
		summary.Metrics = append(summary.Metrics, Metric{
			Name:    "Enterprise value",
			Value:   ev,
			Unit:    "USD",
			Period:  latestMetricPeriod(summary.Metrics),
			Form:    "market cap + debt - cash",
			Concept: "enterpriseValue",
		})
	}

	if latestPrice != nil {
		eps, ok := metricValue(summary.Metrics, "Diluted EPS")
		if !ok {
			eps, ok = metricValue(summary.Metrics, "Basic EPS")
		}
		if ok && eps > 0 {
			summary.Metrics = append(summary.Metrics, Metric{
				Name:    "P/E ratio",
				Value:   latestPrice.Price / eps,
				Unit:    "x",
				Period:  latestPrice.Time.Format("2006-01-02 15:04"),
				Form:    "market close / earnings",
				Concept: "LatestClose/AnnualEarningsPerShareDiluted",
			})
		}
	}
}

func metricValue(metrics []Metric, name string) (float64, bool) {
	for _, metric := range metrics {
		if metric.Name == name {
			return metric.Value, true
		}
	}
	return 0, false
}

func latestMetricPeriod(metrics []Metric) string {
	for _, metric := range metrics {
		if metric.Period != "" {
			return metric.Period
		}
	}
	return ""
}

func AddPERatio(summary *Summary, facts sec.CompanyFacts, latestPrice market.LatestPrice) bool {
	if summary == nil || latestPrice.Price <= 0 {
		return false
	}

	eps, ok := latestAnnualMetric(facts, metricSpec{Name: "Annual diluted EPS", Unit: "USD/shares", Concepts: []string{"EarningsPerShareDiluted"}})
	if !ok || eps.Value <= 0 {
		return false
	}

	summary.Metrics = append(summary.Metrics, Metric{
		Name:    "P/E ratio",
		Value:   latestPrice.Price / eps.Value,
		Unit:    "x",
		Period:  latestPrice.Time.Format("2006-01-02 15:04"),
		Form:    eps.Form,
		Filed:   eps.Filed,
		Concept: "LatestClose/AnnualEarningsPerShareDiluted",
	})
	summary.Sources = append(summary.Sources, Source{
		Name:        latestPrice.SourceName,
		URL:         latestPrice.SourceURL,
		RetrievedAt: latestPrice.RetrievedAt.Format("2006-01-02 15:04:05 MST"),
	})
	return true
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

func latestAnnualMetric(facts sec.CompanyFacts, spec metricSpec) (Metric, bool) {
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
			if unit.Val == nil || !isAnnualForm(unit.Form) {
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

func isAnnualForm(form string) bool {
	switch strings.ToUpper(form) {
	case "10-K", "20-F", "40-F":
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
