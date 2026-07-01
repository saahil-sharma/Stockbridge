package sec

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type companyTickerEntry struct {
	CIK    int    `json:"cik_str"`
	Ticker string `json:"ticker"`
	Title  string `json:"title"`
}

func ParseCompanies(r io.Reader, sourceURL string, retrievedAt time.Time) ([]Company, error) {
	var entries map[string]companyTickerEntry
	if err := json.NewDecoder(r).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode SEC company tickers: %w", err)
	}

	companies := make([]Company, 0, len(entries))
	for _, entry := range entries {
		companies = append(companies, Company{
			CIK:         entry.CIK,
			Ticker:      entry.Ticker,
			Title:       entry.Title,
			SourceURL:   sourceURL,
			RetrievedAt: retrievedAt,
		})
	}
	return companies, nil
}

type CompanyFacts struct {
	CIK         int                      `json:"cik"`
	EntityName  string                   `json:"entityName"`
	Facts       map[string]TaxonomyFacts `json:"facts"`
	SourceURL   string                   `json:"-"`
	RetrievedAt time.Time                `json:"-"`
}

type TaxonomyFacts map[string]ConceptFact

type ConceptFact struct {
	Label       string                `json:"label"`
	Description string                `json:"description"`
	Units       map[string][]FactUnit `json:"units"`
}

type FactUnit struct {
	End   string   `json:"end"`
	Val   *float64 `json:"val"`
	Accn  string   `json:"accn"`
	FY    int      `json:"fy"`
	FP    string   `json:"fp"`
	Form  string   `json:"form"`
	Filed string   `json:"filed"`
	Frame string   `json:"frame"`
}
