# Stockbridge Symbol Universe

Stockbridge resolves ticker identity through layered sources:

1. Nasdaq Trader `nasdaqlisted.txt` for Nasdaq-listed securities.
2. Nasdaq Trader `otherlisted.txt` for NYSE, NYSE American, NYSE Arca, Cboe BZX, and IEX-listed securities.
3. SEC `company_tickers.json` in the analyzer as a reporting-company fallback.
4. `fallback.go`, a small local curated fallback for major ADRs and foreign issuers that can have incomplete or non-standard SEC company-facts coverage.

The live Nasdaq Trader files are the scalable source for listed U.S. common stocks and many U.S.-traded ADRs. The SEC company ticker file broadens coverage to reporting companies and S&P 500 constituents when the exchange directory lookup is unavailable or incomplete.

To refresh the local foreign issuer fallback, update `curatedFallbackListings` in `fallback.go`. Do not add fabricated fundamentals; the fallback only resolves ticker identity. If SEC company facts are missing or do not include supported standardized concepts, Stockbridge should render the company identity with a data-availability note.
