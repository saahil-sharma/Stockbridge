package symbols

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	nasdaqListedURL = "https://www.nasdaqtrader.com/dynamic/SymDir/nasdaqlisted.txt"
	otherListedURL  = "https://www.nasdaqtrader.com/dynamic/SymDir/otherlisted.txt"
)

type Client struct {
	httpClient      *http.Client
	nasdaqListedURL string
	otherListedURL  string
}

type Listing struct {
	Symbol        string
	SecurityName  string
	Exchange      string
	Market        string
	ETF           bool
	TestIssue     bool
	SourceURL     string
	FileCreatedAt string
	RetrievedAt   time.Time
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		httpClient:      httpClient,
		nasdaqListedURL: nasdaqListedURL,
		otherListedURL:  otherListedURL,
	}
}

func (c *Client) Lookup(ctx context.Context, ticker string) (Listing, error) {
	nasdaqListing, found, err := c.lookupNasdaqListed(ctx, ticker)
	if err != nil {
		return Listing{}, err
	}
	if found {
		return nasdaqListing, nil
	}

	otherListing, found, err := c.lookupOtherListed(ctx, ticker)
	if err != nil {
		return Listing{}, err
	}
	if found {
		return otherListing, nil
	}

	return Listing{}, fmt.Errorf("%s was not found in Nasdaq Trader listed symbol directories", ticker)
}

func (c *Client) lookupNasdaqListed(ctx context.Context, ticker string) (Listing, bool, error) {
	body, retrievedAt, err := c.fetch(ctx, c.nasdaqListedURL)
	if err != nil {
		return Listing{}, false, err
	}
	defer body.Close()

	listings, fileCreatedAt, err := ParseNasdaqListed(body, c.nasdaqListedURL, retrievedAt)
	if err != nil {
		return Listing{}, false, err
	}

	for _, listing := range listings {
		if listing.Symbol == ticker {
			if listing.TestIssue {
				return Listing{}, false, fmt.Errorf("%s is marked as a test issue in the Nasdaq-listed symbol directory", ticker)
			}
			listing.FileCreatedAt = fileCreatedAt
			return listing, true, nil
		}
	}

	return Listing{}, false, nil
}

func (c *Client) lookupOtherListed(ctx context.Context, ticker string) (Listing, bool, error) {
	body, retrievedAt, err := c.fetch(ctx, c.otherListedURL)
	if err != nil {
		return Listing{}, false, err
	}
	defer body.Close()

	listings, fileCreatedAt, err := ParseOtherListed(body, c.otherListedURL, retrievedAt)
	if err != nil {
		return Listing{}, false, err
	}

	for _, listing := range listings {
		if listing.Symbol == ticker {
			if listing.TestIssue {
				return Listing{}, false, fmt.Errorf("%s is marked as a test issue in the other-listed symbol directory", ticker)
			}
			listing.FileCreatedAt = fileCreatedAt
			return listing, true, nil
		}
	}

	return Listing{}, false, nil
}

func (c *Client) fetch(ctx context.Context, url string) (io.ReadCloser, time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, time.Time{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("fetch symbol directory %s: %w", url, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, time.Time{}, fmt.Errorf("fetch symbol directory %s: unexpected status %s", url, resp.Status)
	}

	return resp.Body, time.Now(), nil
}

func ParseNasdaqListed(r io.Reader, sourceURL string, retrievedAt time.Time) ([]Listing, string, error) {
	scanner := bufio.NewScanner(r)
	var listings []Listing
	var fileCreatedAt string
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if lineNo == 1 && strings.HasPrefix(line, "Symbol|") {
			continue
		}
		if strings.HasPrefix(line, "File Creation Time:") {
			fileCreatedAt = strings.TrimSpace(strings.Split(line, "|")[0])
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) < 7 {
			return nil, "", fmt.Errorf("parse Nasdaq-listed symbol directory line %d: expected at least 7 fields", lineNo)
		}

		marketCategory := strings.TrimSpace(fields[2])
		listings = append(listings, Listing{
			Symbol:        strings.TrimSpace(fields[0]),
			SecurityName:  strings.TrimSpace(fields[1]),
			Exchange:      "NASDAQ",
			Market:        nasdaqMarketName(marketCategory),
			ETF:           strings.EqualFold(strings.TrimSpace(fields[6]), "Y"),
			TestIssue:     strings.EqualFold(strings.TrimSpace(fields[3]), "Y"),
			SourceURL:     sourceURL,
			FileCreatedAt: fileCreatedAt,
			RetrievedAt:   retrievedAt,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("read Nasdaq-listed symbol directory: %w", err)
	}

	return listings, fileCreatedAt, nil
}

func ParseOtherListed(r io.Reader, sourceURL string, retrievedAt time.Time) ([]Listing, string, error) {
	scanner := bufio.NewScanner(r)
	var listings []Listing
	var fileCreatedAt string
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if lineNo == 1 && strings.HasPrefix(line, "ACT Symbol|") {
			continue
		}
		if strings.HasPrefix(line, "File Creation Time:") {
			fileCreatedAt = strings.TrimSpace(strings.Split(line, "|")[0])
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) < 7 {
			return nil, "", fmt.Errorf("parse symbol directory line %d: expected at least 7 fields", lineNo)
		}

		listings = append(listings, Listing{
			Symbol:        strings.TrimSpace(fields[0]),
			SecurityName:  strings.TrimSpace(fields[1]),
			Exchange:      otherListedExchangeName(strings.TrimSpace(fields[2])),
			ETF:           strings.EqualFold(strings.TrimSpace(fields[4]), "Y"),
			TestIssue:     strings.EqualFold(strings.TrimSpace(fields[6]), "Y"),
			SourceURL:     sourceURL,
			FileCreatedAt: fileCreatedAt,
			RetrievedAt:   retrievedAt,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("read symbol directory: %w", err)
	}

	return listings, fileCreatedAt, nil
}

func nasdaqMarketName(category string) string {
	switch category {
	case "Q":
		return "Nasdaq Global Select Market"
	case "G":
		return "Nasdaq Global Market"
	case "S":
		return "Nasdaq Capital Market"
	default:
		return "Nasdaq"
	}
}

func otherListedExchangeName(code string) string {
	switch code {
	case "A":
		return "NYSE American"
	case "N":
		return "New York Stock Exchange"
	case "P":
		return "NYSE Arca"
	case "Z":
		return "Cboe BZX"
	case "V":
		return "Investors Exchange"
	default:
		return code
	}
}
