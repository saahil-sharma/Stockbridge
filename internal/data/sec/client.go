package sec

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	companyTickersURL = "https://www.sec.gov/files/company_tickers.json"
	companyFactsURL   = "https://data.sec.gov/api/xbrl/companyfacts/CIK%010d.json"
)

type Client struct {
	httpClient *http.Client
	userAgent  string
}

type Company struct {
	CIK         int
	Ticker      string
	Title       string
	SourceURL   string
	RetrievedAt time.Time
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		httpClient: httpClient,
		userAgent:  "Stockbridge CLI contact@example.com",
	}
}

func (c *Client) LookupCompany(ctx context.Context, ticker string) (Company, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, companyTickersURL, nil)
	if err != nil {
		return Company{}, err
	}
	c.applyHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Company{}, fmt.Errorf("fetch SEC company tickers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Company{}, fmt.Errorf("fetch SEC company tickers: unexpected status %s", resp.Status)
	}

	companies, err := ParseCompanies(resp.Body, companyTickersURL, time.Now())
	if err != nil {
		return Company{}, err
	}

	for _, company := range companies {
		if equivalentTickers(company.Ticker, ticker) {
			return company, nil
		}
	}

	return Company{}, fmt.Errorf("%s was not found in SEC company_tickers.json", ticker)
}

func equivalentTickers(left, right string) bool {
	return canonicalTicker(left) == canonicalTicker(right)
}

func canonicalTicker(input string) string {
	ticker := strings.ToUpper(strings.TrimSpace(input))
	ticker = strings.ReplaceAll(ticker, "-", ".")
	return ticker
}

func (c *Client) CompanyFacts(ctx context.Context, cik int) (CompanyFacts, error) {
	sourceURL := fmt.Sprintf(companyFactsURL, cik)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return CompanyFacts{}, err
	}
	c.applyHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return CompanyFacts{}, fmt.Errorf("fetch SEC company facts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CompanyFacts{}, fmt.Errorf("fetch SEC company facts: unexpected status %s", resp.Status)
	}

	var facts CompanyFacts
	if err := json.NewDecoder(resp.Body).Decode(&facts); err != nil {
		return CompanyFacts{}, fmt.Errorf("decode SEC company facts: %w", err)
	}
	facts.SourceURL = sourceURL
	facts.RetrievedAt = time.Now()
	return facts, nil
}

func (c *Client) applyHeaders(req *http.Request) {
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
}
