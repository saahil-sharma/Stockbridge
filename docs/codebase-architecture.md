# Stockbridge Codebase Architecture

This document explains how the current Stockbridge codebase is structured, how data moves through the application, and where the implementation has diverged from the original product plan in `docs/stockbridge-build-map.md` and `AGENTS.md`.

## Current Product Shape

Stockbridge is currently a Go application with two runnable entrypoints:

- `cmd/stockbridge`: a command-line tool built with Cobra.
- `cmd/stockbridge-web`: a local HTTP web app that reuses the same analysis backend.

The primary workflow is still `stockbridge analyze <ticker>`. The command validates and resolves a ticker, fetches company and fundamentals data, normalizes that data into a shared `analysis.Summary`, and renders either styled terminal output or plain text.

The web app is a second frontend over the same analyzer. It accepts a ticker query, renders the fundamentals report in HTML, and attempts to include an interactive historical price chart.

## Top-Level Layout

```text
cmd/
  stockbridge/
    main.go              # CLI binary entrypoint
  stockbridge-web/
    main.go              # local web server binary entrypoint

internal/
  app/
    analyzer.go          # central orchestration service
    analyzer_test.go
  analysis/
    summary.go           # normalized report model and metric extraction
    summary_test.go
  cli/
    root.go              # root Cobra command
    analyze.go           # analyze command and CLI output selection
    about.go             # static project/help text
  data/
    fundamentals/        # Financial Modeling Prep fallback fundamentals client
    market/              # Yahoo Finance chart/latest-close client
    sec/                 # SEC company ticker and company-facts client
    symbols/             # Nasdaq Trader symbol directory and fallback listings
  report/
    styled.go            # ANSI styled terminal report
    text.go              # plain text report
  web/
    server.go            # local HTML app, theming, watchlist, chart rendering

docs/
  stockbridge-build-map.md
  desktop-backend-transition.md
  codebase-architecture.md

go.mod
go.sum
Makefile
AGENTS.md
```

The implementation mostly follows the expected `cmd/`, `internal/`, and `docs/` shape from the project instructions. Some planned directories do not exist yet, especially `internal/news/`, `internal/cache/`, `internal/domain/`, `testdata/`, and dedicated `analysis` files for valuation, risk, news, and stance.

## Binaries And Entrypoints

### CLI: `cmd/stockbridge`

`cmd/stockbridge/main.go` only constructs and executes `cli.NewRootCommand()`.

The actual command tree lives in `internal/cli`:

- `root.go` defines the root `stockbridge` command and registers subcommands.
- `analyze.go` defines `stockbridge analyze <ticker>`.
- `about.go` defines `stockbridge about`.

`stockbridge analyze` accepts:

- `--format styled`, the default terminal view.
- `--format text`, accepted by validation but currently bypassed for stdout because stdout always uses `RenderStyled`.
- `--output, -o <file>`, which writes plain text to a file.

The analyze command constructs an `http.Client` with a 20 second timeout, creates an `app.Analyzer`, calls `Analyze`, and passes the returned `analysis.Summary` to a renderer.

### Web App: `cmd/stockbridge-web`

`cmd/stockbridge-web/main.go` starts an HTTP server at `127.0.0.1:8080` by default. It builds the same `app.Analyzer` used by the CLI and passes it to `web.NewServer`.

The web app currently serves one route, `/`, from `internal/web/server.go`. A request with `?ticker=AMZN` runs the analyzer, renders the fundamentals report, and tries to fetch a price chart with `Analyzer.PriceChart`.

## Central Orchestration: `internal/app`

`internal/app/analyzer.go` is the main application service. It owns four clients:

- `symbols.Client` for ticker validation and listing lookup.
- `sec.Client` for SEC company ticker lookup and company facts.
- `market.Client` for Yahoo Finance latest close and price charts.
- `fundamentals.Client` for Financial Modeling Prep fundamentals fallback.

`NewAnalyzer` wires those clients together. It reads `STOCKBRIDGE_FMP_API_KEY` from the environment and passes it to the fundamentals client. If the key is empty, the market-fundamentals fallback is skipped instead of making unauthenticated provider requests.

### `Analyze` Flow

The current high-level flow is:

```text
raw ticker
  -> symbols.NormalizeTicker
  -> symbols.Client.Lookup
  -> sec.Client.LookupCompany
  -> sec.Client.CompanyFacts
  -> analysis.BuildSummary
  -> market.Client.LatestClose
  -> analysis.AddPERatio
  -> report renderer or web template
```

There are fallback branches:

1. If symbol directory lookup fails but SEC company lookup succeeds, Stockbridge treats the company as an SEC reporting-company fallback.
2. If SEC lookup or SEC company facts are unavailable for a recognized listing, Stockbridge tries Financial Modeling Prep fundamentals through `buildMarketSummary`.
3. If SEC facts exist but no supported standardized metrics can be extracted, Stockbridge also tries Financial Modeling Prep before returning an availability-only report.
4. If all usable fundamentals sources fail, Stockbridge returns a partial report that identifies the company and explains missing data.

`Analyzer.PriceChart` is separate from `Analyze`. It normalizes a ticker and asks the market client for multiple historical price ranges. The CLI does not currently render that chart; the web app does.

## Data Clients

### `internal/data/symbols`

This package validates and resolves ticker listings.

`ticker.go` contains syntax normalization:

- Tickers are uppercased and trimmed.
- Valid tickers match `^[A-Z][A-Z0-9.-]{0,13}$`.
- Dot and dash variants are treated as equivalent for class-share tickers such as `BRK.B` and `BRK-B`.

`client.go` fetches and parses:

- `https://www.nasdaqtrader.com/dynamic/SymDir/nasdaqlisted.txt`
- `https://www.nasdaqtrader.com/dynamic/SymDir/otherlisted.txt`

It returns a `symbols.Listing` with symbol, security name, exchange, market, ETF/test flags, source URL, file timestamp, and retrieval time.

`fallback.go` contains a curated local list of major foreign issuers and ADRs such as `TSM`, `ASML`, `BABA`, `NVO`, and others. This fallback only resolves ticker identity and listing metadata; it does not contain fabricated fundamentals.

### `internal/data/sec`

This package talks to SEC EDGAR APIs:

- `https://www.sec.gov/files/company_tickers.json`
- `https://data.sec.gov/api/xbrl/companyfacts/CIK##########.json`

`LookupCompany` maps ticker to CIK and SEC company title. `CompanyFacts` fetches XBRL company facts for that CIK.

`types.go` defines the minimal SEC data structures used by the analysis layer:

- `Company`
- `CompanyFacts`
- `TaxonomyFacts`
- `ConceptFact`
- `FactUnit`

The analysis code currently reads only `us-gaap` facts and only selected concepts.

### `internal/data/fundamentals`

This package is a newer fallback provider for companies that do not have usable SEC company facts. It calls Financial Modeling Prep endpoints:

- `/profile/{ticker}`
- `/income-statement/{ticker}?period=annual&limit=4`
- `/balance-sheet-statement/{ticker}?period=annual&limit=4`
- `/cash-flow-statement/{ticker}?period=annual&limit=4`

It normalizes provider JSON into:

- `fundamentals.Company`
- `fundamentals.Metric`
- `fundamentals.Source`
- `fundamentals.Snapshot`

It extracts or calculates:

- Revenue
- Revenue growth
- Gross margin
- Operating margin
- Net income
- Net income growth
- Basic EPS
- Diluted EPS
- Assets
- Liabilities
- Stockholders' equity
- Cash and equivalents
- Total debt
- Operating cash flow
- Capital expenditures
- Free cash flow

Valuation metrics that require market cap, debt, cash, equity, and latest close are finished in `analysis.BuildMarketSummary`, not inside the provider client.

### `internal/data/market`

This package uses Yahoo Finance's chart endpoint:

```text
https://query1.finance.yahoo.com/v8/finance/chart/{ticker}
```

It supports these periods:

- `1Y`
- `YTD`
- `1M`
- `5D`
- `1D`

The package has two roles:

- `LatestClose` gives the analyzer a recent price for P/E calculations.
- `FetchBundle` gives the web app historical close series for chart rendering.

Ticker symbols are converted to Yahoo's dash format for class shares, so `BRK.B` becomes `BRK-B`.

## Analysis Model

`internal/analysis/summary.go` defines the shared report model:

```go
type Summary struct {
    CompanyName string
    Ticker      string
    CIK         int
    Listing     symbols.Listing
    Metrics     []Metric
    Sources     []Source
    Notes       []string
}
```

Everything user-facing is ultimately rendered from this model.

### SEC Metric Extraction

`BuildSummary` creates a report from SEC company facts. It looks for the latest useful `10-K`, `10-Q`, `20-F`, or `40-F` facts for these metrics:

- Revenue
- Net income
- Assets
- Liabilities
- Stockholders' equity
- Cash and equivalents
- Operating cash flow
- Capital expenditures
- Basic EPS
- Diluted EPS

Metric selection sorts candidate facts by filed date, then period end date, and uses the newest candidate.

`AddPERatio` adds P/E when a latest close is available and positive annual diluted EPS can be extracted from SEC facts.

### Market Fundamentals Summary

`BuildMarketSummary` adapts `fundamentals.Snapshot` into `analysis.Summary`. It also calculates:

- P/S ratio
- P/B ratio
- Debt/equity ratio
- Enterprise value
- P/E ratio, if latest close and EPS are available

It adds notes that make clear the report is using market-data fundamentals because SEC company facts are unavailable or incomplete.

### Availability Summary

`BuildAvailabilitySummary` creates a partial report when the ticker is recognized but no usable fundamentals data could be fetched. This is the graceful degradation path for data-source failures.

## Report Rendering

### Plain Text

`internal/report/text.go` renders a simple portable report with:

- Header
- Ticker identity and exchange metadata
- Fundamentals table
- Notes
- Sources

It is used for file output through `--output`.

### Styled Terminal Output

`internal/report/styled.go` renders ANSI-colored terminal output. It uses in-repo ANSI escape helpers, not Lip Gloss. The current palette includes deep blue, cyan, green, amber, lavender, paper, muted gray, and panel gray.

This is a deliberate implementation deviation from the project instruction that styled terminal output should use Bubble Tea-compatible models and Lip Gloss styles. The current renderer preserves the styled terminal experience while reducing dependency pressure.

## Web App

`internal/web/server.go` is a self-contained HTML application embedded as a Go string. It includes:

- Ticker search form.
- Company summary.
- Fundamentals table.
- Notes and sources.
- Historical price chart with range tabs.
- Theme selector using `localStorage`.
- Watchlist using `localStorage`.

The web app still talks directly to `app.Analyzer`; there is no JSON API layer yet. This matches the near-term transition idea in `docs/desktop-backend-transition.md`, where the analyzer becomes a reusable backend service and presentation remains separate.

## Testing

Tests currently cover:

- Ticker normalization and invalid ticker rejection.
- Nasdaq Trader file parsing.
- Other-listed exchange parsing.
- Curated foreign issuer fallback lookup.
- Ticker equivalence and variant generation.
- SEC company ticker parsing and class punctuation equivalence.
- Yahoo chart response parsing and ticker formatting.
- SEC summary metric extraction.
- P/E ratio calculation.
- Analyzer behavior for unknown tickers.
- Analyzer fallback to market fundamentals for `TSM` when SEC data is missing.

The tests use mocked `http.Client` transports and inline response bodies instead of live network calls. There is no `testdata/` directory yet, even though the original plan called for fixture files.

The test command used successfully in this workspace was:

```sh
env CGO_ENABLED=0 GOPROXY=off GOSUMDB=off GOCACHE=/private/tmp/go-build GOMODCACHE=/Users/saahilsharma/go/pkg/mod go test ./...
```

`CGO_ENABLED=0` was needed in this environment to avoid runtime issues with generated test binaries.

## Configuration And Secrets

The only current application-specific environment variable is:

```text
STOCKBRIDGE_FMP_API_KEY
```

It is read by `app.NewAnalyzer` and passed to the Financial Modeling Prep client. It is not required at compile time, and the code does not store it in source control.

The original plan also mentioned:

- `STOCKBRIDGE_FINNHUB_API_KEY`
- `STOCKBRIDGE_NEWS_API_KEY`
- `STOCKBRIDGE_CACHE_DIR`

Those are not currently used.

## Dependency Model

The current Go module is intentionally small:

- Direct dependency: `github.com/spf13/cobra`
- Indirect dependencies: Cobra's transitive packages

Earlier styled-output work used Lip Gloss, but the current renderer is plain Go plus ANSI escape sequences. This keeps the binary and tests simpler but diverges from the stated terminal UI instruction to use Lip Gloss.

## Current Data Flow In Practice

### SEC Reporting Company With Supported Facts

```text
CLI or web request
  -> app.Analyzer.Analyze
  -> symbols.NormalizeTicker
  -> symbols.Client.Lookup
  -> sec.Client.LookupCompany
  -> sec.Client.CompanyFacts
  -> analysis.BuildSummary
  -> market.Client.LatestClose
  -> analysis.AddPERatio
  -> report.RenderStyled / report.RenderText / web template
```

### Recognized Foreign Issuer Or Non-SEC Fundamentals Case

```text
CLI or web request
  -> app.Analyzer.Analyze
  -> symbols.NormalizeTicker
  -> symbols.Client.Lookup or curated fallback listing
  -> SEC lookup or SEC facts fail, or SEC facts have no supported metrics
  -> fundamentals.Client.Snapshot
  -> market.Client.LatestClose
  -> analysis.BuildMarketSummary
  -> report.RenderStyled / report.RenderText / web template
```

### Recognized Ticker With No Usable Fundamentals

```text
CLI or web request
  -> ticker and listing recognized
  -> SEC facts unavailable
  -> market fundamentals unavailable
  -> analysis.BuildAvailabilitySummary
  -> report with identity, notes, and sources only
```

## Deviations From The Initial Product Idea

The initial product idea was broader than the current implementation. The codebase has moved in some useful directions and has postponed other planned areas.

### Added Earlier Than Planned

The original build map positioned the CLI as the first interface and mentioned a future desktop/web transition. The current code already includes `cmd/stockbridge-web` and a full local HTML interface with themes, watchlist, and historical chart rendering.

The original plan said market data and valuation ratios should come after SEC fundamentals. The current code now includes a Financial Modeling Prep fallback for foreign issuers and non-SEC cases, plus Yahoo latest close data for P/E calculation.

### Changed From SEC-Only To Hybrid Fundamentals

The first implementation leaned heavily on SEC company facts. That worked for U.S. reporting companies but failed for tickers like `TSM` when standardized SEC facts were unavailable.

The current analyzer now prefers SEC facts when available, then falls back to Financial Modeling Prep fundamentals. This is a major product shift: Stockbridge is no longer only an SEC company-facts renderer. It is becoming a hybrid fundamentals aggregator with explicit source notes.

### Terminal Styling Changed Direction

The project instructions asked for Bubble Tea-compatible models and Lip Gloss styles for styled terminal output. The current styled renderer uses direct ANSI escape sequences instead.

That gives a styled terminal report with fewer dependencies, but it is not Bubble Tea or Lip Gloss based. If the terminal UI becomes interactive later, this should be revisited.

### Report Scope Is Still Narrower Than Planned

The original report target included:

- Business snapshot
- Margins and growth
- Balance sheet strength
- Cash flow
- Valuation
- Recent news
- Risk discussion
- Evidence-backed bullish, neutral, or bearish stance

The current report includes fundamentals, basic valuation metrics, notes, and sources. It does not yet include:

- Recent material news
- News sentiment
- Analyst estimates
- Full risk classification
- Final stance
- Business description in the CLI report
- Forward-looking valuation ratios
- Multi-period trend tables beyond selected growth calculations

### CLI Flags Are Fewer Than Planned

The build map proposed:

```sh
stockbridge analyze IBM --format markdown
stockbridge analyze IBM --news-window 14d
stockbridge analyze IBM --offline --fixture testdata/ibm_bundle.json
stockbridge analyze IBM --no-cache
stockbridge analyze IBM --verbose
```

The current CLI supports only:

```sh
stockbridge analyze IBM
stockbridge analyze IBM --format styled
stockbridge analyze IBM --format text
stockbridge analyze IBM --output reports/IBM.txt
```

There is no markdown output, news window, offline fixture flag, cache control, or verbose mode yet.

### No Cache Layer Yet

The original plan mentioned cache storage and `STOCKBRIDGE_CACHE_DIR`. The current code makes live HTTP requests every time unless tests inject mocked clients. There is no cache package and no persisted source data.

### No Dedicated Domain Model Layer Yet

`docs/desktop-backend-transition.md` proposed a future `internal/domain` package for frontend-neutral report and chart models. The current code still uses `analysis.Summary` as the shared model for CLI and web. That is workable today, but as UI and API needs grow, a clearer domain boundary would help.

### Test Fixtures Are Inline, Not File-Based

The plan called for deterministic fixtures under `testdata/`. The current tests are deterministic, but most fixtures are inline strings or mocked HTTP responses inside test files.

### Web Charting Is Active Despite CLI Chart Deferral

The project instructions say historical chart code is sidelined from the active CLI report and should be preserved for future desktop/web UI. The current code follows that for the CLI, but the web app actively uses chart data and renders it in the browser.

## Current Strengths

- The core analyzer is reusable by CLI and web.
- External facts are source-tracked with retrieval timestamps.
- Ticker validation happens before external analysis calls.
- Recognized tickers degrade gracefully when data is missing.
- SEC and market-data fundamentals are separated into data clients.
- Report rendering is separate from analysis.
- Tests avoid live network calls.
- The new non-SEC fallback path is covered by an analyzer-level test.

## Current Gaps And Risks

- Financial Modeling Prep responses are normalized directly in one provider client; there is no provider interface yet.
- The fallback fundamentals client assumes several provider fields are in USD even though its notes say provider-native currency may apply.
- The CLI accepts `--format text`, but stdout rendering currently always uses styled output unless `--output` is used.
- The Makefile still uses `-ldflags=-linkmode=external`, while tests in this workspace passed with `CGO_ENABLED=0`; the Makefile may not represent the most reliable local test command.
- There is no retry, rate-limit handling, or cache for live data.
- News, risk, stance, and analyst estimates are not implemented.
- The web app is embedded in one large Go string, which makes frontend iteration harder.
- The SEC user agent is still a placeholder: `Stockbridge CLI contact@example.com`.

## Practical Next Steps

The next maintainable steps are:

1. Fix CLI `--format text` so stdout respects the selected format.
2. Document `STOCKBRIDGE_FMP_API_KEY` in a user-facing setup guide.
3. Move inline test responses into `testdata/` once fixtures grow.
4. Add a provider interface around non-SEC fundamentals before adding a second provider.
5. Add cache support before increasing live data usage.
6. Add news only after source attribution and freshness rules are clearly modeled.
7. Split the embedded web template once the web UI becomes a larger product surface.
