package fundamentals

import "time"

type Company struct {
	Name        string
	Ticker      string
	Exchange    string
	Sector      string
	Industry    string
	Description string
	Currency    string
	MarketCap   float64
	SourceURL   string
	RetrievedAt time.Time
}

type Metric struct {
	Name    string
	Value   float64
	Unit    string
	Period  string
	Source  string
	Concept string
}

type Source struct {
	Name        string
	URL         string
	RetrievedAt time.Time
}

type Snapshot struct {
	Company Company
	Metrics []Metric
	Sources []Source
	Notes   []string
}
