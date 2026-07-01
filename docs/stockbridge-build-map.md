# Stockbridge Build Map

Working name: Stockbridge

Purpose: build a CLI that accepts a listed U.S. stock ticker symbol and returns a sourced, current fundamental-analysis report that combines financial statement data, valuation metrics, recent company news, and a clear non-advisory interpretation.

## Recommended Language

Use Go.

Reasons:
- Go produces easy-to-distribute single binaries for macOS, Linux, and Windows.
- The standard library is strong for HTTP clients, JSON parsing, CSV parsing, testing, and command execution.
- Static typing helps keep financial data normalization explicit.
- Go's built-in tooling covers formatting, tests, modules, documentation, and profiling without much setup.
- The project does not need Python's data-science stack at first; the core challenge is reliable data collection, normalization, and report generation.

Tradeoff: Python would be faster for exploratory financial modeling, but Go is a better fit for a polished CLI users can install and run without managing an interpreter or virtual environment.

## Primary Tooling

### CLI Framework

Use Cobra.

Cobra is a Go CLI framework built around commands, arguments, and flags. It supports subcommands, POSIX-style flags, generated help, shell autocomplete, and manpage generation. It is also used by major Go CLIs such as Kubernetes, Hugo, and GitHub CLI.

Source: https://github.com/spf13/cobra

### Configuration

Use environment variables first. Add Viper only if configuration grows beyond a few API keys and flags.

Initial environment variables:
- `STOCKBRIDGE_FMP_API_KEY` for Financial Modeling Prep, if selected.
- `STOCKBRIDGE_FINNHUB_API_KEY` for Finnhub, if selected.
- `STOCKBRIDGE_NEWS_API_KEY` for a paid/general news provider, if selected.
- `STOCKBRIDGE_CACHE_DIR` to override local cache storage.

### Terminal Output

Start with plain terminal output using the Go standard library.

Add these later only if the report needs richer terminal presentation:
- Lip Gloss for styled terminal tables and section formatting.
- Bubble Tea for an interactive terminal UI.
- Bubbles for reusable text inputs, spinners, and viewports.

Bubble Tea is a Go framework for terminal apps and works for simple or complex inline/full-window terminal applications. It is useful, but not necessary for the first working version.

Source: https://github.com/charmbracelet/bubbletea

### Testing

Use Go's built-in test tooling:
- `go test ./...` for unit tests.
- Fixture files under `testdata/` for deterministic SEC, symbol-directory, quote, fundamentals, and news responses.
- Optional `httptest` servers for API client tests.

## Data Sources

### Ticker And NYSE Validation

Use Nasdaq Trader Symbol Directory files to validate listed securities:
- `nasdaqlisted.txt` for Nasdaq-listed securities.
- `otherlisted.txt` for NYSE and other non-Nasdaq exchange-listed securities.

The symbol directory files are updated throughout the day. The `otherlisted.txt` definitions mark `N` as New York Stock Exchange, while `nasdaqlisted.txt` includes Nasdaq market categories.

Sources:
- https://www.nasdaqtrader.com/trader.aspx?id=symboldirdefs
- https://www.nasdaqtrader.com/dynamic/SymDir/nasdaqlisted.txt
- https://www.nasdaqtrader.com/dynamic/SymDir/otherlisted.txt

### SEC Filings And XBRL Financials

Use SEC EDGAR APIs for baseline company facts and filings.

SEC `data.sec.gov` APIs provide JSON-formatted company submissions and XBRL financial-statement data, do not require API keys, and are updated in real time as filings are disseminated.

Source: https://www.sec.gov/search-filings/edgar-application-programming-interfaces

Implementation note: SEC data requires careful mapping from ticker to CIK, then from XBRL company facts to normalized metrics. This should be reliable but not rushed.

### Market Data And Valuation Ratios

Use one provider abstraction and keep providers swappable.

Recommended starting order:
1. SEC data for reported fundamentals.
2. A commercial/free-tier market-data API for price, market cap, enterprise value, and precomputed ratios.
3. Manual calculation from SEC data when provider values are missing or suspect.

Provider candidates:
- Financial Modeling Prep: broad fundamentals, ratios, profiles, and market data.
- Finnhub: company fundamentals, quotes, and market news.
- Polygon.io: strong market data, usually better when real-time/reliable pricing matters.

Do not hard-code analysis logic to one vendor's response shape. Normalize into internal structs first.

### News

Use a news provider with timestamps, URLs, source names, and article summaries.

Minimum fields needed:
- title
- publisher/source
- URL
- published timestamp
- summary or description
- related ticker/company field when available

Provider candidates:
- Finnhub company news
- Financial Modeling Prep stock news
- NewsAPI or a similar general-news provider

Every report should show which news items were used, their publication dates, and the retrieval time.

## Proposed CLI Shape

Initial command:

```sh
stockbridge analyze IBM
```

Useful flags:

```sh
stockbridge analyze IBM --format text
stockbridge analyze IBM --format markdown
stockbridge analyze IBM --output reports/IBM.md
stockbridge analyze IBM --news-window 14d
stockbridge analyze IBM --offline --fixture testdata/ibm_bundle.json
stockbridge analyze IBM --no-cache
stockbridge analyze IBM --verbose
```

Future commands:

```sh
stockbridge sources
stockbridge validate-ticker IBM
stockbridge cache clear
stockbridge explain-methodology
```

## Internal Architecture

```text
cmd/stockbridge/
  main.go

internal/cli/
  root.go
  analyze.go

internal/data/
  symbols/
  sec/
  marketdata/
  news/

internal/analysis/
  fundamentals.go
  valuation.go
  risk.go
  news.go
  stance.go

internal/report/
  model.go
  text.go
  markdown.go

internal/cache/
  cache.go

docs/
  methodology.md
  data-sources.md

testdata/
  symbols/
  sec/
  marketdata/
  news/
```

## Core Data Flow

```text
ticker input
  -> normalize and validate symbol syntax
  -> load current symbol directory
  -> confirm listing exchange
  -> map ticker to SEC CIK
  -> fetch SEC filings/company facts
  -> fetch current market data
  -> fetch recent company news
  -> normalize all provider responses
  -> compute metrics and trend signals
  -> summarize news impact
  -> generate sourced report
```

## Report Sections

Every report should include:
- Header: company name, ticker, exchange, report timestamp, data freshness.
- Business snapshot: sector, industry, description, market cap if available.
- Financial performance: revenue, gross margin, operating margin, net income, EPS trends.
- Balance sheet strength: cash, debt, net debt, liquidity, leverage.
- Cash flow: operating cash flow, free cash flow, capex trend.
- Valuation: P/E, forward P/E if sourced, EV/EBITDA, price/sales, price/book when available.
- Growth and quality: multi-period revenue and earnings growth, return metrics if available.
- Recent news: material headlines with dates, sources, and likely fundamental relevance.
- Risks: business, balance-sheet, valuation, execution, macro, and data-quality risks.
- Stance: bullish, neutral, or bearish only when supported by the evidence.
- Sources: URLs and retrieval timestamps.
- Disclaimer: analysis is informational, not personalized financial advice.

## Build Phases

### Phase 1: Skeleton

Deliver:
- Go module.
- Cobra-based CLI.
- `stockbridge analyze <ticker>`.
- Static fixture-backed report generation.
- Unit tests for ticker validation and report rendering.

### Phase 2: Symbol Validation

Deliver:
- Nasdaq Trader `nasdaqlisted.txt` and `otherlisted.txt` clients.
- Exchange validation.
- Cache with source timestamp.
- Fixture tests for symbol parsing.

### Phase 3: SEC Fundamentals

Deliver:
- SEC ticker-to-CIK support.
- SEC submissions and company-facts clients.
- Normalized financial metric extraction.
- Tests using stored SEC JSON fixtures.

### Phase 4: Market Data Provider

Deliver:
- Provider interface for quote, market cap, enterprise value, and ratios.
- One concrete provider implementation.
- Environment-variable API-key loading.
- Clear missing-data handling.

### Phase 5: News Provider

Deliver:
- News provider interface.
- One concrete provider implementation.
- News-window flag.
- Report section that distinguishes facts from interpretation.

### Phase 6: Analysis Engine

Deliver:
- Metric scoring.
- Trend analysis.
- Risk classification.
- Evidence-backed stance.
- Methodology documentation.

### Phase 7: Output Polish

Deliver:
- Text and Markdown output.
- Optional file output.
- Better terminal tables.
- Shell completion.
- Install/release instructions.

## First Implementation Recommendation

Start with Go, Cobra, SEC EDGAR, Nasdaq Trader symbol validation, and a fixture-backed report path. Add a paid or free-tier market/news API only after the internal report model and provider interfaces are stable.
