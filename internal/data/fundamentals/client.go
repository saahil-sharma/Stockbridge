package fundamentals

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

func NewClient(httpClient *http.Client, apiKey string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    "https://financialmodelingprep.com/api/v3",
		apiKey:     strings.TrimSpace(apiKey),
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.apiKey != ""
}

func (c *Client) Snapshot(ctx context.Context, ticker string) (Snapshot, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return Snapshot{}, fmt.Errorf("ticker is required")
	}
	if !c.Configured() {
		return Snapshot{}, fmt.Errorf("Financial Modeling Prep API key is not configured")
	}

	profile, err := c.profile(ctx, ticker)
	if err != nil {
		return Snapshot{}, err
	}

	profileURL := c.endpointURL("/profile/" + url.PathEscape(ticker))
	retrievedAt := time.Now()
	company := profile.toCompany(ticker)
	company.SourceURL = profileURL
	company.RetrievedAt = retrievedAt
	snapshot := Snapshot{
		Company: company,
		Sources: []Source{
			{Name: "Financial Modeling Prep profile", URL: profileURL, RetrievedAt: retrievedAt},
		},
	}

	incomeStatements, incomeSource, err := c.incomeStatements(ctx, ticker)
	if err != nil {
		snapshot.Notes = append(snapshot.Notes, "Income statement data is temporarily unavailable or rate-limited.")
	} else {
		snapshot.Sources = append(snapshot.Sources, incomeSource)
		appendIncomeMetrics(&snapshot, incomeStatements)
	}

	balanceSheets, balanceSource, err := c.balanceSheets(ctx, ticker)
	if err != nil {
		snapshot.Notes = append(snapshot.Notes, "Balance sheet data is temporarily unavailable or rate-limited.")
	} else {
		snapshot.Sources = append(snapshot.Sources, balanceSource)
		appendBalanceMetrics(&snapshot, balanceSheets)
	}

	cashFlows, cashFlowSource, err := c.cashFlows(ctx, ticker)
	if err != nil {
		snapshot.Notes = append(snapshot.Notes, "Cash flow data is temporarily unavailable or rate-limited.")
	} else {
		snapshot.Sources = append(snapshot.Sources, cashFlowSource)
		appendCashFlowMetrics(&snapshot, cashFlows)
	}

	snapshot.Notes = append(snapshot.Notes, "Market fundamentals are sourced from Financial Modeling Prep and may be reported in the provider's native currency.")
	return snapshot, nil
}

func (c *Client) profile(ctx context.Context, ticker string) (profileResponse, error) {
	var payload []profileResponse
	if err := c.getJSON(ctx, "/profile/"+url.PathEscape(ticker), &payload); err != nil {
		return profileResponse{}, err
	}
	if len(payload) == 0 {
		return profileResponse{}, fmt.Errorf("profile data for %s was empty", ticker)
	}
	return payload[0], nil
}

func (c *Client) incomeStatements(ctx context.Context, ticker string) ([]incomeStatementResponse, Source, error) {
	var payload []incomeStatementResponse
	path := "/income-statement/" + url.PathEscape(ticker) + "?period=annual&limit=4"
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, Source{}, err
	}
	if len(payload) == 0 {
		return nil, Source{}, fmt.Errorf("income statement data for %s was empty", ticker)
	}
	sortStatements(payload, func(item incomeStatementResponse) string { return item.Date })
	return payload, Source{Name: "Financial Modeling Prep income statement", URL: c.endpointURL(path), RetrievedAt: time.Now()}, nil
}

func (c *Client) balanceSheets(ctx context.Context, ticker string) ([]balanceSheetResponse, Source, error) {
	var payload []balanceSheetResponse
	path := "/balance-sheet-statement/" + url.PathEscape(ticker) + "?period=annual&limit=4"
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, Source{}, err
	}
	if len(payload) == 0 {
		return nil, Source{}, fmt.Errorf("balance sheet data for %s was empty", ticker)
	}
	sortStatements(payload, func(item balanceSheetResponse) string { return item.Date })
	return payload, Source{Name: "Financial Modeling Prep balance sheet", URL: c.endpointURL(path), RetrievedAt: time.Now()}, nil
}

func (c *Client) cashFlows(ctx context.Context, ticker string) ([]cashFlowResponse, Source, error) {
	var payload []cashFlowResponse
	path := "/cash-flow-statement/" + url.PathEscape(ticker) + "?period=annual&limit=4"
	if err := c.getJSON(ctx, path, &payload); err != nil {
		return nil, Source{}, err
	}
	if len(payload) == 0 {
		return nil, Source{}, fmt.Errorf("cash flow data for %s was empty", ticker)
	}
	sortStatements(payload, func(item cashFlowResponse) string { return item.Date })
	return payload, Source{Name: "Financial Modeling Prep cash flow statement", URL: c.endpointURL(path), RetrievedAt: time.Now()}, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return err
	}
	q := u.Query()
	if c.apiKey != "" {
		q.Set("apikey", c.apiKey)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Stockbridge CLI")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("fetch fundamentals: %w", ctx.Err())
		}
		return fmt.Errorf("fetch fundamentals: request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fetch fundamentals: unexpected status %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode fundamentals: %w", err)
	}
	return nil
}

func (c *Client) endpointURL(path string) string {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return c.baseURL + path
	}
	q := u.Query()
	q.Del("apikey")
	u.RawQuery = q.Encode()
	return u.String()
}

func sortStatements[T any](items []T, date func(T) string) {
	sort.Slice(items, func(i, j int) bool {
		return date(items[i]) > date(items[j])
	})
}

type profileResponse struct {
	CompanyName string  `json:"companyName"`
	Symbol      string  `json:"symbol"`
	Exchange    string  `json:"exchange"`
	Sector      string  `json:"sector"`
	Industry    string  `json:"industry"`
	Description string  `json:"description"`
	Currency    string  `json:"currency"`
	MarketCap   float64 `json:"mktCap"`
}

func (p profileResponse) toCompany(ticker string) Company {
	name := strings.TrimSpace(p.CompanyName)
	if name == "" {
		name = ticker
	}
	return Company{
		Name:        name,
		Ticker:      strings.ToUpper(strings.TrimSpace(firstNonEmpty(p.Symbol, ticker))),
		Exchange:    strings.TrimSpace(p.Exchange),
		Sector:      strings.TrimSpace(p.Sector),
		Industry:    strings.TrimSpace(p.Industry),
		Description: strings.TrimSpace(p.Description),
		Currency:    strings.TrimSpace(p.Currency),
		MarketCap:   p.MarketCap,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type incomeStatementResponse struct {
	Date            string   `json:"date"`
	Revenue         *float64 `json:"revenue"`
	GrossProfit     *float64 `json:"grossProfit"`
	OperatingIncome *float64 `json:"operatingIncome"`
	NetIncome       *float64 `json:"netIncome"`
	EPS             *float64 `json:"eps"`
	EPSDiluted      *float64 `json:"epsdiluted"`
}

type balanceSheetResponse struct {
	Date                string   `json:"date"`
	CashAndEquivalents  *float64 `json:"cashAndCashEquivalents"`
	CashAndShortTermInv *float64 `json:"cashAndShortTermInvestments"`
	TotalAssets         *float64 `json:"totalAssets"`
	TotalLiabilities    *float64 `json:"totalLiabilities"`
	TotalEquity         *float64 `json:"totalStockholdersEquity"`
	LongTermDebt        *float64 `json:"longTermDebt"`
	ShortTermDebt       *float64 `json:"shortTermDebt"`
	TotalDebt           *float64 `json:"totalDebt"`
}

type cashFlowResponse struct {
	Date               string   `json:"date"`
	OperatingCashFlow  *float64 `json:"operatingCashFlow"`
	CapitalExpenditure *float64 `json:"capitalExpenditure"`
	FreeCashFlow       *float64 `json:"freeCashFlow"`
}

func appendIncomeMetrics(snapshot *Snapshot, statements []incomeStatementResponse) {
	latest, previous, ok := pairLatestStatements(statements, func(item incomeStatementResponse) string { return item.Date })
	if !ok {
		return
	}
	if latest.Revenue != nil {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Revenue", Value: *latest.Revenue, Unit: "USD", Period: latest.Date, Source: "Financial Modeling Prep income statement", Concept: "revenue"})
		if previous.Revenue != nil && *previous.Revenue != 0 {
			snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Revenue growth", Value: ((*latest.Revenue - *previous.Revenue) / math.Abs(*previous.Revenue)) * 100, Unit: "%", Period: latest.Date, Source: "Financial Modeling Prep income statement", Concept: "revenueGrowth"})
		}
	}
	if latest.GrossProfit != nil && latest.Revenue != nil && *latest.Revenue != 0 {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Gross margin", Value: (*latest.GrossProfit / *latest.Revenue) * 100, Unit: "%", Period: latest.Date, Source: "Financial Modeling Prep income statement", Concept: "grossMargin"})
	}
	if latest.OperatingIncome != nil && latest.Revenue != nil && *latest.Revenue != 0 {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Operating margin", Value: (*latest.OperatingIncome / *latest.Revenue) * 100, Unit: "%", Period: latest.Date, Source: "Financial Modeling Prep income statement", Concept: "operatingMargin"})
	}
	if latest.NetIncome != nil {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Net income", Value: *latest.NetIncome, Unit: "USD", Period: latest.Date, Source: "Financial Modeling Prep income statement", Concept: "netIncome"})
		if previous.NetIncome != nil && *previous.NetIncome != 0 {
			snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Net income growth", Value: ((*latest.NetIncome - *previous.NetIncome) / math.Abs(*previous.NetIncome)) * 100, Unit: "%", Period: latest.Date, Source: "Financial Modeling Prep income statement", Concept: "netIncomeGrowth"})
		}
	}
	if latest.EPS != nil {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Basic EPS", Value: *latest.EPS, Unit: "USD/shares", Period: latest.Date, Source: "Financial Modeling Prep income statement", Concept: "eps"})
	}
	if latest.EPSDiluted != nil {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Diluted EPS", Value: *latest.EPSDiluted, Unit: "USD/shares", Period: latest.Date, Source: "Financial Modeling Prep income statement", Concept: "epsdiluted"})
	}
}

func appendBalanceMetrics(snapshot *Snapshot, statements []balanceSheetResponse) {
	latest, _, ok := pairLatestStatements(statements, func(item balanceSheetResponse) string { return item.Date })
	if !ok {
		return
	}
	if latest.TotalAssets != nil {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Assets", Value: *latest.TotalAssets, Unit: "USD", Period: latest.Date, Source: "Financial Modeling Prep balance sheet", Concept: "totalAssets"})
	}
	if latest.TotalLiabilities != nil {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Liabilities", Value: *latest.TotalLiabilities, Unit: "USD", Period: latest.Date, Source: "Financial Modeling Prep balance sheet", Concept: "totalLiabilities"})
	}
	if latest.TotalEquity != nil {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Stockholders' equity", Value: *latest.TotalEquity, Unit: "USD", Period: latest.Date, Source: "Financial Modeling Prep balance sheet", Concept: "totalStockholdersEquity"})
	}
	if cash := cashValue(latest); cash != nil {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Cash and equivalents", Value: *cash, Unit: "USD", Period: latest.Date, Source: "Financial Modeling Prep balance sheet", Concept: "cash"})
	}
	if debt := debtValue(latest); debt != 0 {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Total debt", Value: debt, Unit: "USD", Period: latest.Date, Source: "Financial Modeling Prep balance sheet", Concept: "debt"})
	}
}

func appendCashFlowMetrics(snapshot *Snapshot, statements []cashFlowResponse) {
	latest, _, ok := pairLatestStatements(statements, func(item cashFlowResponse) string { return item.Date })
	if !ok {
		return
	}
	if latest.OperatingCashFlow != nil {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Operating cash flow", Value: *latest.OperatingCashFlow, Unit: "USD", Period: latest.Date, Source: "Financial Modeling Prep cash flow statement", Concept: "operatingCashFlow"})
	}
	if latest.CapitalExpenditure != nil {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Capital expenditures", Value: *latest.CapitalExpenditure, Unit: "USD", Period: latest.Date, Source: "Financial Modeling Prep cash flow statement", Concept: "capitalExpenditure"})
	}
	if latest.FreeCashFlow != nil {
		snapshot.Metrics = append(snapshot.Metrics, Metric{Name: "Free cash flow", Value: *latest.FreeCashFlow, Unit: "USD", Period: latest.Date, Source: "Financial Modeling Prep cash flow statement", Concept: "freeCashFlow"})
	}
}

func pairLatestStatements[T any](items []T, date func(T) string) (T, T, bool) {
	var zero T
	if len(items) == 0 {
		return zero, zero, false
	}
	if len(items) == 1 {
		return items[0], zero, true
	}

	sorted := append([]T(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		return date(sorted[i]) > date(sorted[j])
	})
	return sorted[0], sorted[1], true
}

func cashValue(stmt balanceSheetResponse) *float64 {
	if stmt.CashAndEquivalents != nil {
		return stmt.CashAndEquivalents
	}
	return stmt.CashAndShortTermInv
}

func debtValue(stmt balanceSheetResponse) float64 {
	if stmt.TotalDebt != nil {
		return *stmt.TotalDebt
	}
	total := 0.0
	if stmt.LongTermDebt != nil {
		total += *stmt.LongTermDebt
	}
	if stmt.ShortTermDebt != nil {
		total += *stmt.ShortTermDebt
	}
	return total
}
