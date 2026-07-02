package market

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

type Period string

const (
	PeriodOneYear  Period = "1Y"
	PeriodYTD      Period = "YTD"
	PeriodOneMonth Period = "1M"
	PeriodFiveDays Period = "5D"
	PeriodOneDay   Period = "1D"
)

var DefaultPeriods = []Period{
	PeriodOneYear,
	PeriodYTD,
	PeriodOneMonth,
	PeriodFiveDays,
	PeriodOneDay,
}

type Point struct {
	Time  time.Time
	Close float64
}

type Series struct {
	Period      Period
	Points      []Point
	SourceURL   string
	RetrievedAt time.Time
}

type LatestPrice struct {
	Ticker      string
	Price       float64
	Time        time.Time
	SourceName  string
	SourceURL   string
	RetrievedAt time.Time
}

type Bundle struct {
	Ticker      string
	Series      map[Period]Series
	SourceName  string
	SourceURL   string
	RetrievedAt time.Time
}

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    "https://query1.finance.yahoo.com/v8/finance/chart",
	}
}

func (c *Client) FetchBundle(ctx context.Context, ticker string) (Bundle, error) {
	bundle := Bundle{
		Ticker:     ticker,
		Series:     make(map[Period]Series, len(DefaultPeriods)),
		SourceName: "Yahoo Finance chart API",
	}

	for _, period := range DefaultPeriods {
		series, err := c.FetchSeries(ctx, ticker, period)
		if err != nil {
			return Bundle{}, err
		}
		bundle.Series[period] = series
		if bundle.SourceURL == "" {
			bundle.SourceURL = series.SourceURL
			bundle.RetrievedAt = series.RetrievedAt
		}
	}

	return bundle, nil
}

func (c *Client) LatestClose(ctx context.Context, ticker string) (LatestPrice, error) {
	series, err := c.FetchSeries(ctx, ticker, PeriodOneDay)
	if err != nil {
		return LatestPrice{}, err
	}
	if len(series.Points) == 0 {
		return LatestPrice{}, fmt.Errorf("latest close for %s was not available", ticker)
	}

	point := series.Points[len(series.Points)-1]
	return LatestPrice{
		Ticker:      ticker,
		Price:       point.Close,
		Time:        point.Time,
		SourceName:  "Yahoo Finance chart API",
		SourceURL:   series.SourceURL,
		RetrievedAt: series.RetrievedAt,
	}, nil
}

func (c *Client) FetchSeries(ctx context.Context, ticker string, period Period) (Series, error) {
	queryRange, interval, err := yahooRange(period)
	if err != nil {
		return Series{}, err
	}

	u, err := url.Parse(c.baseURL + "/" + yahooTicker(ticker))
	if err != nil {
		return Series{}, err
	}
	q := u.Query()
	q.Set("range", queryRange)
	q.Set("interval", interval)
	q.Set("includePrePost", "false")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Series{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Stockbridge CLI")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Series{}, fmt.Errorf("fetch chart data for %s %s: %w", ticker, period, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Series{}, fmt.Errorf("fetch chart data for %s %s: unexpected status %s", ticker, period, resp.Status)
	}

	points, err := ParseYahooChart(resp.Body)
	if err != nil {
		return Series{}, fmt.Errorf("parse chart data for %s %s: %w", ticker, period, err)
	}
	if len(points) == 0 {
		return Series{}, fmt.Errorf("chart data for %s %s did not include close prices", ticker, period)
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Time.Before(points[j].Time)
	})

	return Series{
		Period:      period,
		Points:      points,
		SourceURL:   u.String(),
		RetrievedAt: time.Now(),
	}, nil
}

func yahooRange(period Period) (string, string, error) {
	switch period {
	case PeriodOneYear:
		return "1y", "1d", nil
	case PeriodYTD:
		return "ytd", "1d", nil
	case PeriodOneMonth:
		return "1mo", "1d", nil
	case PeriodFiveDays:
		return "5d", "30m", nil
	case PeriodOneDay:
		return "1d", "5m", nil
	default:
		return "", "", fmt.Errorf("unsupported chart period %q", period)
	}
}

func yahooTicker(ticker string) string {
	return strings.ReplaceAll(strings.ToUpper(ticker), ".", "-")
}

type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []*float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error any `json:"error"`
	} `json:"chart"`
}

func ParseYahooChart(r interface{ Read([]byte) (int, error) }) ([]Point, error) {
	var payload yahooChartResponse
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Chart.Error != nil {
		return nil, fmt.Errorf("provider returned chart error: %v", payload.Chart.Error)
	}
	if len(payload.Chart.Result) == 0 {
		return nil, nil
	}

	result := payload.Chart.Result[0]
	if len(result.Indicators.Quote) == 0 {
		return nil, nil
	}

	closes := result.Indicators.Quote[0].Close
	points := make([]Point, 0, min(len(result.Timestamp), len(closes)))
	for i := 0; i < len(result.Timestamp) && i < len(closes); i++ {
		if closes[i] == nil || math.IsNaN(*closes[i]) || math.IsInf(*closes[i], 0) {
			continue
		}
		points = append(points, Point{
			Time:  time.Unix(result.Timestamp[i], 0),
			Close: *closes[i],
		})
	}

	return points, nil
}
