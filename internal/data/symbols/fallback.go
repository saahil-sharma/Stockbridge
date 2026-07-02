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
	{Symbol: "AZN", SecurityName: "AstraZeneca PLC American Depositary Shares", Exchange: "NASDAQ", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "GSK", SecurityName: "GSK plc American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "UL", SecurityName: "Unilever PLC American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "DEO", SecurityName: "Diageo plc American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "PDD", SecurityName: "PDD Holdings Inc. American Depositary Shares", Exchange: "NASDAQ", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "JD", SecurityName: "JD.com, Inc. American Depositary Shares", Exchange: "NASDAQ", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "BIDU", SecurityName: "Baidu, Inc. American Depositary Shares", Exchange: "NASDAQ", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "INFY", SecurityName: "Infosys Limited American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "HDB", SecurityName: "HDFC Bank Limited American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "TTM", SecurityName: "Tata Motors Limited American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "RACE", SecurityName: "Ferrari N.V. Common Shares", Exchange: "New York Stock Exchange", Market: "Foreign ordinary shares", SourceURL: curatedFallbackSource},
	{Symbol: "LYG", SecurityName: "Lloyds Banking Group plc American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "UBS", SecurityName: "UBS Group AG Registered Ordinary Shares", Exchange: "New York Stock Exchange", Market: "Foreign ordinary shares", SourceURL: curatedFallbackSource},
	{Symbol: "NVS", SecurityName: "Novartis AG American Depositary Shares", Exchange: "New York Stock Exchange", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "SNY", SecurityName: "Sanofi American Depositary Shares", Exchange: "NASDAQ", Market: "ADR", SourceURL: curatedFallbackSource},
	{Symbol: "SHOP", SecurityName: "Shopify Inc. Class A Subordinate Voting Shares", Exchange: "NASDAQ", Market: "Foreign ordinary shares", SourceURL: curatedFallbackSource},
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
