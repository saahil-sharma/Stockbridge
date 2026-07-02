package symbols

import "time"

const curatedFallbackSource = "local curated foreign issuer and ADR fallback"

var curatedFallbackListings = []Listing{
	{Symbol: "TSM", SecurityName: "Taiwan Semiconductor Manufacturing Company Limited American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "ASML", SecurityName: "ASML Holding N.V. New York Registry Shares", Exchange: "NASDAQ", Market: "Foreign ordinary shares", SourceURL: curatedFallbackSource},
	{Symbol: "BABA", SecurityName: "Alibaba Group Holding Limited American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "TCEHY", SecurityName: "Tencent Holdings Limited Unsponsored ADR", Exchange: "OTC", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "SAP", SecurityName: "SAP SE American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "SONY", SecurityName: "Sony Group Corporation American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "TM", SecurityName: "Toyota Motor Corporation American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "NVO", SecurityName: "Novo Nordisk A/S American Depositary Receipts", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "HSBC", SecurityName: "HSBC Holdings plc American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "BP", SecurityName: "BP p.l.c. American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "SHEL", SecurityName: "Shell plc American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "RIO", SecurityName: "Rio Tinto plc American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
}

func CuratedFallbackListing(ticker string) (Listing, bool) {
	for _, listing := range curatedFallbackListings {
		if EquivalentTickers(listing.Symbol, ticker) {
			listing.RetrievedAt = time.Now()
			return listing, true
		}
	}
	return Listing{}, false
}
